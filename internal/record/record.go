// Package record reads a messy hours CSV and separates rows a person can
// trust from rows a person has to review.
//
// A row is either clean or an exception. This package does not drop rows
// and does not pick a winner when two rows claim the same employee and month.
package record

// RequiredColumns are the only columns the audit reads.
// Extra columns are reported as warnings.
var RequiredColumns = []string{
	"employee_id",
	"name",
	"hire_date",
	"month",
	"hours_worked",
	"coverage_offered",
}

// FullTimeHours is the IRS monthly equivalent of 30 hours of service per week.
// Under the monthly measurement method, an employee with at least this many
// hours of service in a calendar month is full-time for the employer shared
// responsibility provisions. This demo does not implement the look-back
// measurement method.
//
// Source: https://www.irs.gov/affordable-care-act/employers/identifying-full-time-employees
const FullTimeHours = 130

// MaxMonthHours is 31 days times 24 hours. It is a data-quality bound for this
// demo, not an IRS rule. Values above it are held for review instead of capped.
const MaxMonthHours = 744

// MaxDataRows is the most data rows AuditCSV will accept.
// A larger file returns an error instead of a truncated result.
const MaxDataRows = 10000

// Raw is one source row, still strings, after surrounding whitespace is trimmed.
type Raw struct {
	EmployeeID      string `json:"employee_id"`
	Name            string `json:"name"`
	HireDate        string `json:"hire_date"`
	Month           string `json:"month"`
	HoursWorked     string `json:"hours_worked"`
	CoverageOffered string `json:"coverage_offered"`
}

// Input is a raw row plus the line it came from.
// For a CSV, Row is the 1-based physical line of the record. The header is line 1.
// For JSON, Row is whatever the client sent. A non-positive Row is replaced
// with the 1-based position in that request.
type Input struct {
	Row int `json:"row"`
	Raw
}

// Row is a record that passed every validation rule.
type Row struct {
	Row             int     `json:"row"`
	EmployeeID      string  `json:"employee_id"`
	Name            string  `json:"name"`
	HireDate        string  `json:"hire_date"`
	Month           string  `json:"month"`
	HoursWorked     float64 `json:"hours_worked"`
	CoverageOffered bool    `json:"coverage_offered"`
	FullTime        bool    `json:"full_time"`
	CoverageGap     bool    `json:"coverage_gap"`
}

// Exception is one problem on one source row.
// A row with two problems produces two exceptions and stays out of Clean.
type Exception struct {
	Row    int    `json:"row"`
	Field  string `json:"field"`
	Reason string `json:"reason"`
	Raw    Raw    `json:"raw"`
}

// Summary counts the batch. Spreadsheet exports recompute the same
// measures with formulas instead of copying these numbers.
type Summary struct {
	InputRows      int     `json:"input_rows"`
	CleanRows      int     `json:"clean_rows"`
	ExceptionCount int     `json:"exception_count"`
	RowsWithErrors int     `json:"rows_with_errors"`
	FullTimeMonths int     `json:"full_time_months"`
	PartTimeMonths int     `json:"part_time_months"`
	CoverageGaps   int     `json:"coverage_gaps"`
	TotalHours     float64 `json:"total_hours"`
}

// Result is the whole audit of one batch.
type Result struct {
	Clean      []Row       `json:"clean"`
	Exceptions []Exception `json:"exceptions"`
	Summary    Summary     `json:"summary"`
	Warnings   []string    `json:"warnings"`
}
