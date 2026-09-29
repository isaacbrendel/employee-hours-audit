package record

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestMessyFixture(t *testing.T) {
	body, err := os.ReadFile("../../testdata/employees_messy.csv")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(body, []byte{0xEF, 0xBB, 0xBF}) {
		t.Fatal("fixture is missing its UTF-8 BOM")
	}
	if !bytes.Contains(body, []byte("\r\n")) {
		t.Fatal("fixture is missing CRLF")
	}

	got, err := AuditCSV(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}

	if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "Department") {
		t.Fatalf("warnings %v", got.Warnings)
	}

	clean := map[string]Row{}
	for _, row := range got.Clean {
		if _, ok := clean[row.EmployeeID]; ok {
			t.Fatalf("duplicate clean id %s", row.EmployeeID)
		}
		clean[row.EmployeeID] = row
	}

	wantClean := []string{
		"E001", "E002", "E003", "E004", "E005", "E006", "00042",
		"E008", "E009", "E010", "E011", "E012", "E013", "E014",
		"E015", "E016", "E017", "E018",
	}
	if len(clean) != len(wantClean) {
		t.Fatalf("clean ids %v", keys(clean))
	}
	for _, id := range wantClean {
		if _, ok := clean[id]; !ok {
			t.Fatalf("missing clean row %s; have %v", id, keys(clean))
		}
	}

	if clean["E001"].Row != 2 || clean["E001"].FullTime != true || clean["E001"].CoverageGap {
		t.Fatalf("%+v", clean["E001"])
	}
	if !clean["E002"].FullTime || !clean["E002"].CoverageGap || clean["E002"].CoverageOffered {
		t.Fatalf("gap %+v", clean["E002"])
	}
	if clean["E003"].Name != "Alan Turing, Jr." || clean["E003"].CoverageOffered {
		t.Fatalf("%+v", clean["E003"])
	}
	if clean["E004"].FullTime || clean["E004"].HoursWorked != 129.5 || clean["E004"].HireDate != "2015-03-15" {
		t.Fatalf("%+v", clean["E004"])
	}
	if !clean["E005"].FullTime || clean["E005"].HireDate != "2012-03-15" || clean["E005"].HoursWorked != 130 {
		t.Fatalf("%+v", clean["E005"])
	}
	if clean["E006"].Name != "Mary Jackson" || clean["E006"].HoursWorked != 100 || clean["E006"].CoverageOffered {
		t.Fatalf("%+v", clean["E006"])
	}
	if clean["00042"].EmployeeID != "00042" {
		t.Fatalf("%+v", clean["00042"])
	}
	if clean["E009"].Name != "=1+1" || clean["E010"].Name != "@SUM(1,1)" || clean["E011"].Name != "+1+1" {
		t.Fatalf("injection names did not stay intact: %+v %+v %+v", clean["E009"], clean["E010"], clean["E011"])
	}
	if clean["E012"].HireDate != "2024-02-29" || clean["E013"].HoursWorked != 0 || clean["E013"].FullTime {
		t.Fatalf("leap/zero %+v %+v", clean["E012"], clean["E013"])
	}
	if clean["E014"].HoursWorked != 744 || !clean["E014"].FullTime {
		t.Fatalf("%+v", clean["E014"])
	}
	if clean["E015"].EmployeeID != "E015" {
		t.Fatalf("%+v", clean["E015"])
	}
	if clean["E018"].CoverageOffered != true || !clean["E018"].FullTime {
		t.Fatalf("%+v", clean["E018"])
	}

	byID := map[string][]Exception{}
	var blank []Exception
	for _, ex := range got.Exceptions {
		if ex.Raw.EmployeeID == "" && ex.Raw.Name == "" {
			blank = append(blank, ex)
			continue
		}
		byID[ex.Raw.EmployeeID] = append(byID[ex.Raw.EmployeeID], ex)
	}

	one := func(id, field, reason string) {
		t.Helper()
		items := byID[id]
		if len(items) != 1 || items[0].Field != field || !strings.Contains(items[0].Reason, reason) {
			t.Fatalf("%s: %+v", id, items)
		}
		if _, ok := clean[id]; ok {
			t.Fatalf("%s was also clean", id)
		}
	}

	one("", "employee_id", "required") // filled below; missing-id row has a name
	missing := exceptionsNamed(got.Exceptions, "Missing Id")
	if len(missing) != 1 || missing[0].Field != "employee_id" {
		t.Fatalf("missing id %+v", missing)
	}
	one("E020", "name", "required")
	one("E021", "hire_date", "required")
	one("E022", "month", "required")
	one("E023", "hours_worked", "required")
	one("E024", "coverage_offered", "required")
	one("E025", "hire_date", "not an accepted format")
	one("E026", "hire_date", "not an accepted format")
	one("E027", "month", "YYYY-MM")
	one("E028", "month", "YYYY-MM")
	one("E029", "month", "YYYY-MM")
	one("E030", "hours_worked", "plain decimal")
	one("E031", "hours_worked", "cannot be negative")
	one("E032", "hours_worked", "above 744")
	one("E033", "hours_worked", "plain decimal")
	one("E034", "hours_worked", "plain decimal")
	one("E035", "coverage_offered", "yes or no")
	one("E036", "coverage_offered", "yes or no")

	dups := byID["E037"]
	if len(dups) != 2 {
		t.Fatalf("dups %+v", dups)
	}
	for _, ex := range dups {
		if ex.Field != "employee_id" || !strings.Contains(ex.Reason, "choose which record") {
			t.Fatalf("%+v", ex)
		}
	}
	if _, ok := clean["E037"]; ok {
		t.Fatal("duplicate was kept")
	}

	one("E039", "hire_date", "after the reported month")
	one("E040", "row", "were not mapped")
	one("E041", "hours_worked", "required")
	one("E042", "name", "required")
	one("E043", "hire_date", "not an accepted format")

	two := byID["E044"]
	if len(two) != 2 {
		t.Fatalf("%+v", two)
	}
	one("E046", "name", "control character")

	if len(blank) != 6 {
		t.Fatalf("blank record exceptions: %+v", blank)
	}

	want := Summary{
		InputRows:      46,
		CleanRows:      18,
		ExceptionCount: 34,
		RowsWithErrors: 28,
		FullTimeMonths: 5,
		PartTimeMonths: 13,
		CoverageGaps:   1,
		TotalHours:     1816,
	}
	if got.Summary != want {
		t.Fatalf("summary %+v", got.Summary)
	}
}

func exceptionsNamed(items []Exception, name string) []Exception {
	var out []Exception
	for _, item := range items {
		if item.Raw.Name == name {
			out = append(out, item)
		}
	}
	return out
}

func keys(m map[string]Row) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
