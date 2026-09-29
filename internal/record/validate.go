package record

import (
	"fmt"
	"math"
	"slices"
	"strconv"
	"strings"
	"time"
)

// dateLayouts is the whole list. A value that matches none of them is an
// exception. Slash dates are month/day/year because this demo is for US
// reporting. 13/01/2024 does not get a second interpretation as 13 January.
var dateLayouts = []string{
	"2006-01-02",
	"2006/01/02",
	"01/02/2006",
	"1/2/2006",
	"02-Jan-2006",
}

func audit(rows []Input, warnings []string) Result {
	type candidate struct {
		in  Input
		row Row
	}
	var cands []candidate
	var exceptions []Exception

	for _, in := range rows {
		in.Raw = trimRaw(in.Raw)
		issues := validate(in)
		if len(issues) > 0 {
			exceptions = append(exceptions, issues...)
			continue
		}
		cands = append(cands, candidate{in: in, row: toRow(in)})
	}

	groups := map[string][]int{}
	for _, c := range cands {
		k := c.row.EmployeeID + "\x00" + c.row.Month
		groups[k] = append(groups[k], c.row.Row)
	}

	clean := make([]Row, 0, len(cands))
	for _, c := range cands {
		k := c.row.EmployeeID + "\x00" + c.row.Month
		others := groups[k]
		if len(others) > 1 {
			exceptions = append(exceptions, Exception{
				Row:    c.row.Row,
				Field:  "employee_id",
				Reason: duplicateReason(c.row.EmployeeID, c.row.Month, others, c.row.Row),
				Raw:    c.in.Raw,
			})
			continue
		}
		clean = append(clean, c.row)
	}

	if exceptions == nil {
		exceptions = []Exception{}
	}
	sortExceptions(exceptions)
	if warnings == nil {
		warnings = []string{}
	}
	return Result{
		Clean:      clean,
		Exceptions: exceptions,
		Warnings:   warnings,
		Summary:    summarize(len(rows), clean, exceptions),
	}
}

func validate(in Input) []Exception {
	var out []Exception
	add := func(field, reason string) {
		out = append(out, Exception{Row: in.Row, Field: field, Reason: reason, Raw: in.Raw})
	}

	if in.EmployeeID == "" {
		add("employee_id", "employee_id is required")
	} else if ctrl := controlReason(in.EmployeeID); ctrl != "" {
		add("employee_id", "employee_id "+ctrl)
	}

	if in.Name == "" {
		add("name", "name is required")
	} else if ctrl := controlReason(in.Name); ctrl != "" {
		add("name", "name "+ctrl)
	}

	_, hireOK := parseDate(in.HireDate)
	if in.HireDate == "" {
		add("hire_date", "hire_date is required")
	} else if !hireOK {
		add("hire_date", fmt.Sprintf(
			"hire date %q is not an accepted format (YYYY-MM-DD, YYYY/MM/DD, MM/DD/YYYY, M/D/YYYY, or DD-Mon-YYYY)",
			in.HireDate,
		))
	}

	monthOK := validMonth(in.Month)
	if in.Month == "" {
		add("month", "month is required")
	} else if !monthOK {
		add("month", fmt.Sprintf("month %q must be YYYY-MM", in.Month))
	}

	switch {
	case in.HoursWorked == "":
		add("hours_worked", "hours_worked is required")
	case strings.HasPrefix(in.HoursWorked, "-") && plainDecimal(in.HoursWorked[1:]):
		add("hours_worked", fmt.Sprintf("hours cannot be negative (%s)", in.HoursWorked))
	case !plainDecimal(in.HoursWorked):
		add("hours_worked", fmt.Sprintf("hours %q must be a plain decimal number such as 160 or 37.5", in.HoursWorked))
	default:
		hours, err := strconv.ParseFloat(in.HoursWorked, 64)
		if err != nil || math.IsNaN(hours) || math.IsInf(hours, 0) || hours > MaxMonthHours {
			add("hours_worked", fmt.Sprintf(
				"hours %s are above %d, the number of hours in a 31-day month, so the value was not capped or rounded",
				in.HoursWorked, MaxMonthHours,
			))
		}
	}

	if in.CoverageOffered == "" {
		add("coverage_offered", "coverage_offered is required")
	} else if _, ok := parseCoverage(in.CoverageOffered); !ok {
		add("coverage_offered", fmt.Sprintf(
			"coverage %q must be yes or no (also accepted: true/false, y/n, 1/0)",
			in.CoverageOffered,
		))
	}

	if hireOK && monthOK && hiredAfterMonth(mustDate(in.HireDate), in.Month) {
		add("hire_date", fmt.Sprintf("hire date %s is after the reported month %s", in.HireDate, in.Month))
	}
	return out
}

func toRow(in Input) Row {
	hire := mustDate(in.HireDate)
	hours, _ := strconv.ParseFloat(in.HoursWorked, 64)
	coverage, _ := parseCoverage(in.CoverageOffered)
	fullTime := hours >= FullTimeHours
	return Row{
		Row:             in.Row,
		EmployeeID:      in.EmployeeID,
		Name:            in.Name,
		HireDate:        hire.Format("2006-01-02"),
		Month:           in.Month,
		HoursWorked:     hours,
		CoverageOffered: coverage,
		FullTime:        fullTime,
		CoverageGap:     fullTime && !coverage,
	}
}

func parseDate(s string) (time.Time, bool) {
	for _, layout := range dateLayouts {
		t, err := time.Parse(layout, s)
		if err != nil {
			continue
		}
		// Reject values Go would roll into another calendar day.
		if t.Format(layout) != s {
			continue
		}
		return t, true
	}
	return time.Time{}, false
}

func mustDate(s string) time.Time {
	t, _ := parseDate(s)
	return t
}

func validMonth(s string) bool {
	if len(s) != 7 || s[4] != '-' {
		return false
	}
	for i := 0; i < len(s); i++ {
		if i == 4 {
			continue
		}
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	month, _ := strconv.Atoi(s[5:])
	year, _ := strconv.Atoi(s[:4])
	return year >= 1 && month >= 1 && month <= 12
}

func plainDecimal(s string) bool {
	if s == "" {
		return false
	}
	dot := false
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '.' {
			if dot || i == 0 || i == len(s)-1 {
				return false
			}
			dot = true
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}

func parseCoverage(s string) (bool, bool) {
	switch strings.ToLower(s) {
	case "true", "t", "yes", "y", "1":
		return true, true
	case "false", "f", "no", "n", "0":
		return false, true
	default:
		return false, false
	}
}

func hiredAfterMonth(hire time.Time, month string) bool {
	year, _ := strconv.Atoi(month[:4])
	m, _ := strconv.Atoi(month[5:])
	if hire.Year() != year {
		return hire.Year() > year
	}
	return int(hire.Month()) > m
}

func controlReason(s string) string {
	for _, r := range s {
		if r < 0x20 || r == 0x7f {
			return "contains a control character (a line break or a tab, for example)"
		}
	}
	return ""
}

func duplicateReason(id, month string, rows []int, self int) string {
	var others []string
	for _, row := range rows {
		if row != self {
			others = append(others, strconv.Itoa(row))
		}
	}
	where := "row " + others[0]
	if len(others) > 1 {
		where = "rows " + strings.Join(others, ", ")
	}
	return fmt.Sprintf(
		"employee %s is also reported for %s on %s. This row was held so a person can choose which record to keep",
		id, month, where,
	)
}

func sortExceptions(items []Exception) {
	slices.SortStableFunc(items, func(a, b Exception) int {
		if a.Row != b.Row {
			return a.Row - b.Row
		}
		if a.Field != b.Field {
			if a.Field < b.Field {
				return -1
			}
			return 1
		}
		if a.Reason < b.Reason {
			return -1
		}
		if a.Reason > b.Reason {
			return 1
		}
		return 0
	})
}

func summarize(inputRows int, clean []Row, exceptions []Exception) Summary {
	var hours float64
	fullTime := 0
	gaps := 0
	for _, row := range clean {
		hours += row.HoursWorked
		if row.FullTime {
			fullTime++
		}
		if row.CoverageGap {
			gaps++
		}
	}
	seen := map[int]struct{}{}
	for _, ex := range exceptions {
		seen[ex.Row] = struct{}{}
	}
	return Summary{
		InputRows:      inputRows,
		CleanRows:      len(clean),
		ExceptionCount: len(exceptions),
		RowsWithErrors: len(seen),
		FullTimeMonths: fullTime,
		PartTimeMonths: len(clean) - fullTime,
		CoverageGaps:   gaps,
		TotalHours:     hours,
	}
}
