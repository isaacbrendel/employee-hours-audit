package record

import (
	"strings"
	"testing"
)

func TestValidationRules(t *testing.T) {
	base := Input{
		Row: 2,
		Raw: Raw{
			EmployeeID:      "E001",
			Name:            "Ada Lovelace",
			HireDate:        "2020-01-15",
			Month:           "2024-01",
			HoursWorked:     "160",
			CoverageOffered: "yes",
		},
	}

	tests := []struct {
		name       string
		change     func(*Input)
		wantClean  bool
		wantField  string
		wantReason string
		fullTime   bool
		gap        bool
		hire       string
		hours      float64
		coverage   bool
	}{
		{name: "full time with coverage", wantClean: true, fullTime: true, hire: "2020-01-15", hours: 160, coverage: true},
		{
			name:      "full time without coverage is a flag, not an exception",
			change:    func(in *Input) { in.CoverageOffered = "no" },
			wantClean: true, fullTime: true, gap: true, hire: "2020-01-15", hours: 160,
		},
		{
			name:      "just under 130 is part time",
			change:    func(in *Input) { in.HoursWorked = "129.5" },
			wantClean: true, hire: "2020-01-15", hours: 129.5, coverage: true,
		},
		{
			name:      "exactly 130 is full time",
			change:    func(in *Input) { in.HoursWorked = "130" },
			wantClean: true, fullTime: true, hire: "2020-01-15", hours: 130, coverage: true,
		},
		{
			name:      "zero hours is valid",
			change:    func(in *Input) { in.HoursWorked = "0" },
			wantClean: true, hire: "2020-01-15", hours: 0, coverage: true,
		},
		{
			name:      "744 hours is the last accepted value",
			change:    func(in *Input) { in.HoursWorked = "744" },
			wantClean: true, fullTime: true, hire: "2020-01-15", hours: 744, coverage: true,
		},
		{
			name:      "above 744 is not capped",
			change:    func(in *Input) { in.HoursWorked = "744.1" },
			wantField: "hours_worked", wantReason: "above 744",
		},
		{
			name:      "negative hours",
			change:    func(in *Input) { in.HoursWorked = "-4" },
			wantField: "hours_worked", wantReason: "cannot be negative",
		},
		{
			name:      "negative zero",
			change:    func(in *Input) { in.HoursWorked = "-0" },
			wantField: "hours_worked", wantReason: "cannot be negative",
		},
		{
			name:      "thousands separator",
			change:    func(in *Input) { in.HoursWorked = "1,200" },
			wantField: "hours_worked", wantReason: "plain decimal",
		},
		{
			name:      "scientific notation",
			change:    func(in *Input) { in.HoursWorked = "1e2" },
			wantField: "hours_worked", wantReason: "plain decimal",
		},
		{
			name:      "hours text",
			change:    func(in *Input) { in.HoursWorked = "abc" },
			wantField: "hours_worked", wantReason: "plain decimal",
		},
		{
			name:      "missing employee id",
			change:    func(in *Input) { in.EmployeeID = "   " },
			wantField: "employee_id", wantReason: "required",
		},
		{
			name:      "missing name",
			change:    func(in *Input) { in.Name = "" },
			wantField: "name", wantReason: "required",
		},
		{
			name:      "missing hire date",
			change:    func(in *Input) { in.HireDate = "" },
			wantField: "hire_date", wantReason: "required",
		},
		{
			name:      "missing month",
			change:    func(in *Input) { in.Month = "" },
			wantField: "month", wantReason: "required",
		},
		{
			name:      "missing hours",
			change:    func(in *Input) { in.HoursWorked = "" },
			wantField: "hours_worked", wantReason: "required",
		},
		{
			name:      "missing coverage",
			change:    func(in *Input) { in.CoverageOffered = "" },
			wantField: "coverage_offered", wantReason: "required",
		},
		{
			name:      "prose date",
			change:    func(in *Input) { in.HireDate = "March 5th 2024" },
			wantField: "hire_date", wantReason: "not an accepted format",
		},
		{
			name:      "day greater than 12 is not reread as DMY",
			change:    func(in *Input) { in.HireDate = "13/01/2024" },
			wantField: "hire_date", wantReason: "not an accepted format",
		},
		{
			name:      "impossible calendar day",
			change:    func(in *Input) { in.HireDate = "02/31/2024" },
			wantField: "hire_date", wantReason: "not an accepted format",
		},
		{
			name:      "slash month is not YYYY-MM",
			change:    func(in *Input) { in.Month = "2024/01" },
			wantField: "month", wantReason: "YYYY-MM",
		},
		{
			name:      "month 13",
			change:    func(in *Input) { in.Month = "2024-13" },
			wantField: "month", wantReason: "YYYY-MM",
		},
		{
			name:      "compact month",
			change:    func(in *Input) { in.Month = "202401" },
			wantField: "month", wantReason: "YYYY-MM",
		},
		{
			name:      "coverage maybe",
			change:    func(in *Input) { in.CoverageOffered = "maybe" },
			wantField: "coverage_offered", wantReason: "yes or no",
		},
		{
			name:      "coverage offered is not a boolean",
			change:    func(in *Input) { in.CoverageOffered = "offered" },
			wantField: "coverage_offered", wantReason: "yes or no",
		},
		{
			name: "hired after the reported month",
			change: func(in *Input) {
				in.HireDate = "2024-02-01"
				in.Month = "2024-01"
			},
			wantField: "hire_date", wantReason: "after the reported month",
		},
		{
			name: "hired during the reported month",
			change: func(in *Input) {
				in.HireDate = "2024-01-31"
				in.Month = "2024-01"
			},
			wantClean: true, fullTime: true, hire: "2024-01-31", hours: 160, coverage: true,
		},
		{
			name:      "control character in the name",
			change:    func(in *Input) { in.Name = "Ann\tBell" },
			wantField: "name", wantReason: "control character",
		},
		{
			name:      "US slash date normalizes",
			change:    func(in *Input) { in.HireDate = "03/15/2015" },
			wantClean: true, fullTime: true, hire: "2015-03-15", hours: 160, coverage: true,
		},
		{
			name:      "unpadded slash date normalizes",
			change:    func(in *Input) { in.HireDate = "1/2/2020" },
			wantClean: true, fullTime: true, hire: "2020-01-02", hours: 160, coverage: true,
		},
		{
			name:      "year slash date normalizes",
			change:    func(in *Input) { in.HireDate = "2020/01/02" },
			wantClean: true, fullTime: true, hire: "2020-01-02", hours: 160, coverage: true,
		},
		{
			name:      "day month name date normalizes",
			change:    func(in *Input) { in.HireDate = "15-Mar-2012" },
			wantClean: true, fullTime: true, hire: "2012-03-15", hours: 160, coverage: true,
		},
		{
			name: "leap day",
			change: func(in *Input) {
				in.HireDate = "2024-02-29"
				in.Month = "2024-03"
			},
			wantClean: true, fullTime: true, hire: "2024-02-29", hours: 160, coverage: true,
		},
		{
			name:      "leading zeros survive",
			change:    func(in *Input) { in.EmployeeID = "00042" },
			wantClean: true, fullTime: true, hire: "2020-01-15", hours: 160, coverage: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			in := base
			if tt.change != nil {
				tt.change(&in)
			}
			got := AuditRows([]Input{in})
			if tt.wantClean {
				if len(got.Clean) != 1 || len(got.Exceptions) != 0 {
					t.Fatalf("clean=%d exceptions=%+v", len(got.Clean), got.Exceptions)
				}
				row := got.Clean[0]
				if row.FullTime != tt.fullTime || row.CoverageGap != tt.gap {
					t.Fatalf("full_time=%v gap=%v", row.FullTime, row.CoverageGap)
				}
				if row.HireDate != tt.hire || row.HoursWorked != tt.hours || row.CoverageOffered != tt.coverage {
					t.Fatalf("row %+v", row)
				}
				if tt.name == "leading zeros survive" && row.EmployeeID != "00042" {
					t.Fatalf("id %q", row.EmployeeID)
				}
				if tt.name == "zero hours is valid" && (row.HoursWorked != 0 || row.FullTime) {
					t.Fatalf("zero hours stored as %+v", row)
				}
				return
			}
			if len(got.Clean) != 0 {
				t.Fatalf("row was kept: %+v", got.Clean)
			}
			if !hasIssue(got.Exceptions, tt.wantField, tt.wantReason) {
				t.Fatalf("exceptions %+v, want field %s reason %q", got.Exceptions, tt.wantField, tt.wantReason)
			}
		})
	}
}

func TestCoverageSynonyms(t *testing.T) {
	trueValues := []string{"true", "TRUE", "t", "yes", "YES", "y", "1"}
	falseValues := []string{"false", "False", "f", "no", "N", "0"}
	for _, value := range trueValues {
		got := auditOne(Raw{CoverageOffered: value})
		if len(got.Clean) != 1 || !got.Clean[0].CoverageOffered {
			t.Fatalf("%q -> %+v", value, got)
		}
	}
	for _, value := range falseValues {
		got := auditOne(Raw{CoverageOffered: value})
		if len(got.Clean) != 1 || got.Clean[0].CoverageOffered {
			t.Fatalf("%q -> %+v", value, got)
		}
	}
}

func TestSeveralProblemsOnOneRow(t *testing.T) {
	got := auditOne(Raw{
		EmployeeID:      "E044",
		Name:            "Two Problems",
		HireDate:        "2020-01-01",
		Month:           "2024-11",
		HoursWorked:     "nope",
		CoverageOffered: "perhaps",
	})
	if len(got.Exceptions) != 2 {
		t.Fatalf("got %+v", got.Exceptions)
	}
	if got.Exceptions[0].Field != "coverage_offered" || got.Exceptions[1].Field != "hours_worked" {
		t.Fatalf("order %+v", got.Exceptions)
	}
}

func TestDuplicatesAreAllHeld(t *testing.T) {
	rows := []Input{
		{Row: 5, Raw: validRaw("E037", "2024-09", "100", "yes")},
		{Row: 9, Raw: validRaw("E037", "2024-09", "110", "no")},
		{Row: 4, Raw: validRaw("e037", "2024-09", "80", "yes")},
		{Row: 6, Raw: validRaw("E037", "2024-10", "80", "yes")},
	}
	got := AuditRows(rows)
	if len(got.Clean) != 2 {
		t.Fatalf("clean %+v", got.Clean)
	}
	if len(got.Exceptions) != 2 {
		t.Fatalf("exceptions %+v", got.Exceptions)
	}
	if !strings.Contains(got.Exceptions[0].Reason, "row 9") || !strings.Contains(got.Exceptions[1].Reason, "row 5") {
		t.Fatalf("reasons %+v", got.Exceptions)
	}
	for _, ex := range got.Exceptions {
		if ex.Field != "employee_id" {
			t.Fatalf("field %s", ex.Field)
		}
	}
}

func TestIdenticalDuplicatesAreStillHeld(t *testing.T) {
	raw := validRaw("E037", "2024-09", "100", "yes")
	got := AuditRows([]Input{{Row: 2, Raw: raw}, {Row: 3, Raw: raw}})
	if len(got.Clean) != 0 || len(got.Exceptions) != 2 {
		t.Fatalf("%+v", got)
	}
}

func TestWhitespaceIsTrimmedNotRejected(t *testing.T) {
	got := auditOne(Raw{
		EmployeeID:      " E015 ",
		Name:            "  Mary Jackson  ",
		HireDate:        "2016-04-01",
		Month:           "2024-03",
		HoursWorked:     " 100 ",
		CoverageOffered: " no ",
	})
	if len(got.Clean) != 1 {
		t.Fatalf("%+v", got.Exceptions)
	}
	row := got.Clean[0]
	if row.EmployeeID != "E015" || row.Name != "Mary Jackson" || row.HoursWorked != 100 || row.CoverageOffered {
		t.Fatalf("%+v", row)
	}
}

func TestJSONRowNumberFallback(t *testing.T) {
	got := AuditRows([]Input{{Raw: validRaw("E001", "2024-01", "10", "yes")}})
	if len(got.Clean) != 1 || got.Clean[0].Row != 1 {
		t.Fatalf("%+v", got.Clean)
	}
}

func hasIssue(items []Exception, field, reason string) bool {
	for _, item := range items {
		if item.Field == field && strings.Contains(item.Reason, reason) {
			return true
		}
	}
	return false
}

func validRaw(id, month, hours, coverage string) Raw {
	return Raw{
		EmployeeID:      id,
		Name:            "Pat",
		HireDate:        "2020-01-01",
		Month:           month,
		HoursWorked:     hours,
		CoverageOffered: coverage,
	}
}

func auditOne(raw Raw) Result {
	raw = mergeRaw(raw)
	return AuditRows([]Input{{Row: 2, Raw: raw}})
}

func mergeRaw(raw Raw) Raw {
	base := validRaw("E001", "2024-01", "160", "yes")
	if raw.EmployeeID != "" {
		base.EmployeeID = raw.EmployeeID
	}
	if raw.Name != "" {
		base.Name = raw.Name
	}
	if raw.HireDate != "" {
		base.HireDate = raw.HireDate
	}
	if raw.Month != "" {
		base.Month = raw.Month
	}
	if raw.HoursWorked != "" {
		base.HoursWorked = raw.HoursWorked
	}
	if raw.CoverageOffered != "" {
		base.CoverageOffered = raw.CoverageOffered
	}
	return base
}
