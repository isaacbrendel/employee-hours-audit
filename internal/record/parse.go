package record

import (
	"bytes"
	"encoding/csv"
	"errors"
	"fmt"
	"io"
	"strings"
	"unicode/utf8"
)

// AuditCSV reads a whole CSV. The file has to be valid UTF-8.
// A header is required. Blank lines are not records.
// A row whose column count does not match the header is an exception
// and is not mapped into another row's fields.
// A quote error stops the file: the following bytes are no longer aligned,
// and continuing would invent records.
func AuditCSV(r io.Reader) (Result, error) {
	body, err := io.ReadAll(r)
	if err != nil {
		return Result{}, fmt.Errorf("read csv: %w", err)
	}
	body = bytes.TrimPrefix(body, []byte{0xEF, 0xBB, 0xBF})
	if len(bytes.TrimSpace(body)) == 0 {
		return Result{}, errors.New("csv file is empty")
	}
	if !utf8.Valid(body) {
		return Result{}, errors.New("csv file is not valid UTF-8")
	}

	reader := csv.NewReader(bytes.NewReader(body))
	reader.FieldsPerRecord = 0
	header, err := reader.Read()
	if err != nil {
		return Result{}, fmt.Errorf("read header: %w", err)
	}

	index, warnings, err := mapHeader(header)
	if err != nil {
		return Result{}, err
	}
	width := len(header)

	var rows []Input
	var structural []Exception
	for {
		if len(rows)+len(structural) >= MaxDataRows {
			return Result{}, fmt.Errorf("csv has more than %d data rows", MaxDataRows)
		}
		rec, err := reader.Read()
		if err == io.EOF {
			break
		}
		line := lineOf(reader, rec)
		if err != nil && !isFieldCount(err) {
			if line == 0 {
				return Result{}, fmt.Errorf("read csv: %w", err)
			}
			return Result{}, fmt.Errorf("csv record starting on line %d: %w", line, err)
		}
		raw := rawFrom(rec, index)
		if len(rec) != width {
			structural = append(structural, Exception{
				Row:   line,
				Field: "row",
				Reason: fmt.Sprintf(
					"this row has %d columns and the header has %d, so the columns were not mapped",
					len(rec), width,
				),
				Raw: raw,
			})
			continue
		}
		rows = append(rows, Input{Row: line, Raw: raw})
	}

	res := audit(rows, warnings)
	if len(structural) > 0 {
		res.Exceptions = append(res.Exceptions, structural...)
		sortExceptions(res.Exceptions)
	}
	res.Summary = summarize(len(rows)+len(structural), res.Clean, res.Exceptions)
	return res, nil
}

// AuditRows validates a batch that did not come from a CSV.
// The batch is the whole population: duplicates are judged inside it.
func AuditRows(rows []Input) Result {
	ready := make([]Input, len(rows))
	for i, row := range rows {
		if row.Row <= 0 {
			row.Row = i + 1
		}
		row.Raw = trimRaw(row.Raw)
		ready[i] = row
	}
	return audit(ready, nil)
}

func mapHeader(header []string) (map[string]int, []string, error) {
	index := make(map[string]int, len(header))
	var warnings []string
	var problems []string
	seen := make(map[string]int, len(header))

	for i, name := range header {
		original := strings.TrimSpace(name)
		canon := canonicalHeader(original)
		if canon == "" {
			problems = append(problems, "header has an empty column name")
			continue
		}
		if prev, ok := seen[canon]; ok {
			problems = append(problems, fmt.Sprintf("header column %s appears twice (columns %d and %d)", canon, prev+1, i+1))
			continue
		}
		seen[canon] = i
		index[canon] = i
	}

	for _, col := range RequiredColumns {
		if _, ok := index[col]; !ok {
			problems = append(problems, fmt.Sprintf("missing required column %q", col))
		}
	}
	if len(problems) > 0 {
		return nil, nil, errors.New(strings.Join(problems, "; "))
	}

	required := make(map[string]bool, len(RequiredColumns))
	for _, col := range RequiredColumns {
		required[col] = true
	}
	for _, name := range header {
		original := strings.TrimSpace(name)
		canon := canonicalHeader(original)
		if canon != "" && !required[canon] {
			warnings = append(warnings, fmt.Sprintf("ignored extra column %q", original))
		}
	}
	if warnings == nil {
		warnings = []string{}
	}
	return index, warnings, nil
}

func canonicalHeader(s string) string {
	s = strings.TrimPrefix(s, "\uFEFF")
	s = strings.TrimSpace(s)
	s = strings.ToLower(s)
	return strings.ReplaceAll(s, " ", "_")
}

func rawFrom(rec []string, index map[string]int) Raw {
	take := func(column string) string {
		i, ok := index[column]
		if !ok || i >= len(rec) {
			return ""
		}
		return strings.TrimSpace(rec[i])
	}
	return Raw{
		EmployeeID:      take("employee_id"),
		Name:            take("name"),
		HireDate:        take("hire_date"),
		Month:           take("month"),
		HoursWorked:     take("hours_worked"),
		CoverageOffered: take("coverage_offered"),
	}
}

func trimRaw(raw Raw) Raw {
	raw.EmployeeID = strings.TrimSpace(raw.EmployeeID)
	raw.Name = strings.TrimSpace(raw.Name)
	raw.HireDate = strings.TrimSpace(raw.HireDate)
	raw.Month = strings.TrimSpace(raw.Month)
	raw.HoursWorked = strings.TrimSpace(raw.HoursWorked)
	raw.CoverageOffered = strings.TrimSpace(raw.CoverageOffered)
	return raw
}

func isFieldCount(err error) bool {
	var pe *csv.ParseError
	return errors.As(err, &pe) && errors.Is(pe.Err, csv.ErrFieldCount)
}

func lineOf(r *csv.Reader, rec []string) int {
	if len(rec) == 0 {
		return 0
	}
	line, _ := r.FieldPos(0)
	return line
}
