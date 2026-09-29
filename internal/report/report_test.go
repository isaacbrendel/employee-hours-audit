package report

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
	"github.com/xuri/excelize/v2"
)

func TestSafeText(t *testing.T) {
	tests := []struct {
		in, want string
	}{
		{"Ada", "Ada"},
		{"", ""},
		{"=1+1", "'=1+1"},
		{"+1+1", "'+1+1"},
		{"-4", "'-4"},
		{"@SUM(1,1)", "'@SUM(1,1)"},
		{"\t=1+1", "'\t=1+1"},
		{"＝1+1", "'＝1+1"},
		{"130", "130"},
	}
	for _, tt := range tests {
		if got := SafeText(tt.in); got != tt.want {
			t.Fatalf("%q -> %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestWorkbook(t *testing.T) {
	body, err := os.ReadFile("../../testdata/employees_messy.csv")
	if err != nil {
		t.Fatal(err)
	}
	res, err := record.AuditCSV(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	if err := Write(&buf, res); err != nil {
		t.Fatal(err)
	}
	f, err := excelize.OpenReader(bytes.NewReader(buf.Bytes()))
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()

	sheets := f.GetSheetList()
	for _, name := range []string{"Summary", "Clean Data", "Exceptions", "Pivot"} {
		if !contains(sheets, name) {
			t.Fatalf("sheets %v missing %s", sheets, name)
		}
	}

	last := len(res.Clean) + 1
	checks := map[string]string{
		"B6":  fmt.Sprintf("COUNTA('Clean Data'!$A$2:$A$%d)", last),
		"B7":  fmt.Sprintf("COUNTIF('Clean Data'!$G$2:$G$%d,TRUE)", last),
		"B8":  fmt.Sprintf("COUNTIF('Clean Data'!$G$2:$G$%d,FALSE)", last),
		"B9":  fmt.Sprintf("COUNTIF('Clean Data'!$H$2:$H$%d,TRUE)", last),
		"B10": fmt.Sprintf("SUM('Clean Data'!$E$2:$E$%d)", last),
		"B11": fmt.Sprintf("COUNTA(Exceptions!$A$2:$A$%d)", len(res.Exceptions)+1),
	}
	for cell, want := range checks {
		got, err := f.GetCellFormula(sheetSummary, cell)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Fatalf("%s formula %q, want %q", cell, got, want)
		}
	}
	submitted, err := f.GetCellValue(sheetSummary, "B5")
	if err != nil {
		t.Fatal(err)
	}
	if submitted != fmt.Sprint(res.Summary.InputRows) {
		t.Fatalf("rows submitted %q", submitted)
	}
	typedFormula, err := f.GetCellFormula(sheetSummary, "B5")
	if err != nil {
		t.Fatal(err)
	}
	if typedFormula != "" {
		t.Fatalf("rows submitted was stored as a formula: %s", typedFormula)
	}

	panes, err := f.GetPanes(sheetClean)
	if err != nil {
		t.Fatal(err)
	}
	if !panes.Freeze || panes.YSplit != 1 {
		t.Fatalf("panes %+v", panes)
	}

	// E001 is the first clean row: full-time formula, not a cached boolean.
	full, err := f.GetCellFormula(sheetClean, "G2")
	if err != nil || full != "E2>=130" {
		t.Fatalf("full-time formula %q %v", full, err)
	}
	gap, err := f.GetCellFormula(sheetClean, "H2")
	if err != nil || gap != "AND(G2=TRUE,F2=FALSE)" {
		t.Fatalf("gap formula %q %v", gap, err)
	}

	name, err := f.GetCellValue(sheetClean, "B10")
	if err != nil {
		t.Fatal(err)
	}
	// E009 is clean row 9 in file order among the 18, which is excel row 10.
	if name != "'=1+1" {
		t.Fatalf("formula name cell %q", name)
	}
	if formula, _ := f.GetCellFormula(sheetClean, "B10"); formula != "" {
		t.Fatalf("name was stored as a formula: %s", formula)
	}

	foundNegative := false
	rows, err := f.GetRows(sheetExceptions)
	if err != nil {
		t.Fatal(err)
	}
	for _, row := range rows {
		for _, cell := range row {
			if cell == "'-4" {
				foundNegative = true
			}
			if cell == "" {
				continue
			}
			r := []rune(cell)[0]
			switch r {
			case '=', '+', '-', '@', '＝', '＋', '－', '＠':
				t.Fatalf("text cell starts with a formula character: %q", cell)
			}
		}
	}
	if !foundNegative {
		t.Fatal("exception sheet is missing the neutralized -4 hours")
	}

	pivots, err := f.GetPivotTables(sheetPivot)
	if err != nil {
		t.Fatal(err)
	}
	if len(pivots) != 1 {
		t.Fatalf("pivot tables: %d", len(pivots))
	}

	zr, err := zip.NewReader(bytes.NewReader(buf.Bytes()), int64(buf.Len()))
	if err != nil {
		t.Fatal(err)
	}
	hasChart := false
	for _, file := range zr.File {
		if strings.HasPrefix(file.Name, "xl/charts/") {
			hasChart = true
		}
	}
	if !hasChart {
		t.Fatal("workbook has no chart part")
	}
}

func TestEmptyWorkbook(t *testing.T) {
	f, err := Build(record.Result{})
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	for _, cell := range []string{"B6", "B7", "B10", "B11"} {
		got, err := f.GetCellFormula(sheetSummary, cell)
		if err != nil || got != "0" {
			t.Fatalf("%s formula %q %v", cell, got, err)
		}
	}
	note, err := f.GetCellValue(sheetPivot, "A3")
	if err != nil || !strings.Contains(note, "no pivot") {
		t.Fatalf("pivot note %q %v", note, err)
	}
}

func contains(items []string, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}
