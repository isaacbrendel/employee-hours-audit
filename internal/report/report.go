package report

import (
	"fmt"
	"io"
	"slices"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
	"github.com/xuri/excelize/v2"
)

const (
	sheetSummary    = "Summary"
	sheetClean      = "Clean Data"
	sheetExceptions = "Exceptions"
	sheetPivot      = "Pivot"
)

// FullTimeHours is repeated in the worksheet formula so the workbook
// does not depend on a number typed into the summary.
const fullTimeFormulaThreshold = record.FullTimeHours

// Write builds the workbook and copies it to w.
func Write(w io.Writer, res record.Result) error {
	f, err := Build(res)
	if err != nil {
		return err
	}
	_, err = f.WriteTo(w)
	closeErr := f.Close()
	if err != nil {
		return fmt.Errorf("write workbook: %w", err)
	}
	return closeErr
}

// Build returns an unclosed workbook. The caller closes it.
func Build(res record.Result) (*excelize.File, error) {
	f := excelize.NewFile()
	if err := populate(f, res); err != nil {
		f.Close()
		return nil, err
	}
	return f, nil
}

func populate(f *excelize.File, res record.Result) error {
	if err := f.SetSheetName("Sheet1", sheetSummary); err != nil {
		return err
	}
	for _, name := range []string{sheetClean, sheetExceptions, sheetPivot} {
		if _, err := f.NewSheet(name); err != nil {
			return err
		}
	}
	styles, err := makeStyles(f)
	if err != nil {
		return err
	}
	fullCalc := true
	if err := f.SetCalcProps(&excelize.CalcPropsOptions{FullCalcOnLoad: &fullCalc}); err != nil {
		return err
	}
	if err := f.SetDocProps(&excelize.DocProperties{
		Title:       "Employee hours audit",
		Creator:     "employee-hours-audit",
		Description: "Demo only. Not ACA compliance advice.",
	}); err != nil {
		return err
	}
	if err := writeClean(f, styles, res.Clean); err != nil {
		return err
	}
	if err := writeExceptions(f, styles, res.Exceptions); err != nil {
		return err
	}
	if err := writeSummary(f, styles, res); err != nil {
		return err
	}
	if err := writePivot(f, res.Clean); err != nil {
		return err
	}
	if idx, err := f.GetSheetIndex(sheetSummary); err == nil {
		f.SetActiveSheet(idx)
	}
	return nil
}

type styles struct {
	header, title, note, hours, typed int
}

func makeStyles(f *excelize.File) (styles, error) {
	var s styles
	var err error
	s.header, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Bold: true, Color: "FFFFFF", Family: "Calibri"},
		Fill:      excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"1F3A4D"}},
		Alignment: &excelize.Alignment{Vertical: "center"},
	})
	if err != nil {
		return s, err
	}
	s.title, err = f.NewStyle(&excelize.Style{
		Font: &excelize.Font{Bold: true, Size: 18, Color: "1F3A4D"},
	})
	if err != nil {
		return s, err
	}
	s.note, err = f.NewStyle(&excelize.Style{
		Font:      &excelize.Font{Color: "3D4A57", Size: 10},
		Alignment: &excelize.Alignment{WrapText: true, Vertical: "center"},
	})
	if err != nil {
		return s, err
	}
	s.hours, err = f.NewStyle(&excelize.Style{NumFmt: 2})
	if err != nil {
		return s, err
	}
	s.typed, err = f.NewStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"EEF1F4"}},
		Font: &excelize.Font{Color: "1F3A4D"},
		Border: []excelize.Border{
			{Type: "left", Color: "C5CED6", Style: 1},
			{Type: "right", Color: "C5CED6", Style: 1},
			{Type: "top", Color: "C5CED6", Style: 1},
			{Type: "bottom", Color: "C5CED6", Style: 1},
		},
	})
	return s, err
}

func writeClean(f *excelize.File, s styles, rows []record.Row) error {
	header := []string{
		"Employee ID", "Name", "Hire date", "Month", "Hours worked",
		"Coverage offered", "Full time", "Coverage gap",
	}
	for i, name := range header {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(sheetClean, cell, name); err != nil {
			return err
		}
	}
	if err := f.SetCellStyle(sheetClean, "A1", "H1", s.header); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheetClean, 1, 22); err != nil {
		return err
	}

	for i, row := range rows {
		excelRow := i + 2
		values := []struct {
			col string
			val string
		}{
			{"A", row.EmployeeID},
			{"B", row.Name},
			{"C", row.HireDate},
			{"D", row.Month},
		}
		for _, item := range values {
			if err := f.SetCellStr(sheetClean, fmt.Sprintf("%s%d", item.col, excelRow), SafeText(item.val)); err != nil {
				return err
			}
		}
		if err := f.SetCellFloat(sheetClean, fmt.Sprintf("E%d", excelRow), row.HoursWorked, 2, 64); err != nil {
			return err
		}
		if err := f.SetCellBool(sheetClean, fmt.Sprintf("F%d", excelRow), row.CoverageOffered); err != nil {
			return err
		}
		// The sheet decides full-time status. The Go flag is not copied here.
		fullTime := fmt.Sprintf("E%d>=%d", excelRow, fullTimeFormulaThreshold)
		gap := fmt.Sprintf("AND(G%d=TRUE,F%d=FALSE)", excelRow, excelRow)
		if err := f.SetCellFormula(sheetClean, fmt.Sprintf("G%d", excelRow), fullTime); err != nil {
			return err
		}
		if err := f.SetCellFormula(sheetClean, fmt.Sprintf("H%d", excelRow), gap); err != nil {
			return err
		}
	}

	last := 1
	if len(rows) > 0 {
		last = len(rows) + 1
		if err := f.SetCellStyle(sheetClean, "E2", fmt.Sprintf("E%d", last), s.hours); err != nil {
			return err
		}
		if err := markFullTime(f, last); err != nil {
			return err
		}
		if err := f.AutoFilter(sheetClean, fmt.Sprintf("A1:H%d", last), nil); err != nil {
			return err
		}
	}
	widths := []float64{16, 28, 14, 12, 16, 20, 14, 16}
	for i, width := range widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheetClean, col, col, width); err != nil {
			return err
		}
	}
	return f.SetPanes(sheetClean, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
		Selection: []excelize.Selection{{
			SQRef: "A2", ActiveCell: "A2", Pane: "bottomLeft",
		}},
	})
}

func markFullTime(f *excelize.File, last int) error {
	full, err := f.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F4E6D0"}},
	})
	if err != nil {
		return err
	}
	gap, err := f.NewConditionalStyle(&excelize.Style{
		Fill: excelize.Fill{Type: "pattern", Pattern: 1, Color: []string{"F6D0CB"}},
		Font: &excelize.Font{Color: "7A1F1F"},
	})
	if err != nil {
		return err
	}
	area := fmt.Sprintf("A2:H%d", last)
	if err := f.SetConditionalFormat(sheetClean, area, []excelize.ConditionalFormatOptions{{
		Type: "formula", Criteria: "=$G2=TRUE", Format: &full,
	}}); err != nil {
		return err
	}
	return f.SetConditionalFormat(sheetClean, fmt.Sprintf("H2:H%d", last), []excelize.ConditionalFormatOptions{{
		Type: "formula", Criteria: "=$H2=TRUE", Format: &gap, StopIfTrue: true,
	}})
}

func writeExceptions(f *excelize.File, s styles, items []record.Exception) error {
	header := []string{"Row", "Field", "Reason", "Employee ID", "Name", "Hire date", "Month", "Hours worked", "Coverage offered"}
	for i, name := range header {
		cell, err := excelize.CoordinatesToCellName(i+1, 1)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(sheetExceptions, cell, name); err != nil {
			return err
		}
	}
	if err := f.SetCellStyle(sheetExceptions, "A1", "I1", s.header); err != nil {
		return err
	}
	for i, item := range items {
		excelRow := i + 2
		if err := f.SetCellInt(sheetExceptions, fmt.Sprintf("A%d", excelRow), int64(item.Row)); err != nil {
			return err
		}
		texts := []string{item.Field, item.Reason, item.Raw.EmployeeID, item.Raw.Name, item.Raw.HireDate, item.Raw.Month, item.Raw.HoursWorked, item.Raw.CoverageOffered}
		for col, text := range texts {
			cell, err := excelize.CoordinatesToCellName(col+2, excelRow)
			if err != nil {
				return err
			}
			if err := f.SetCellStr(sheetExceptions, cell, SafeText(text)); err != nil {
				return err
			}
		}
	}
	widths := []float64{10, 22, 88, 16, 28, 22, 14, 16, 20}
	for i, width := range widths {
		col, err := excelize.ColumnNumberToName(i + 1)
		if err != nil {
			return err
		}
		if err := f.SetColWidth(sheetExceptions, col, col, width); err != nil {
			return err
		}
	}
	return f.SetPanes(sheetExceptions, &excelize.Panes{
		Freeze:      true,
		YSplit:      1,
		TopLeftCell: "A2",
		ActivePane:  "bottomLeft",
		Selection: []excelize.Selection{{
			SQRef: "A2", ActiveCell: "A2", Pane: "bottomLeft",
		}},
	})
}

func writeSummary(f *excelize.File, s styles, res record.Result) error {
	if err := f.SetCellStr(sheetSummary, "A1", "Employee hours audit"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetSummary, "A1", "A1", s.title); err != nil {
		return err
	}
	note := "Demo only. This workbook is not tax, legal, or ACA filing advice and it is not a Form 1094-C or 1095-C. " +
		"Full time means at least 130 hours in the month, the IRS monthly equivalent of 30 hours of service per week under the monthly measurement method. " +
		"It does not implement look-back measurement, affordability, or a payment under section 4980H. " +
		"https://www.irs.gov/affordable-care-act/employers/identifying-full-time-employees"
	if err := f.SetCellStr(sheetSummary, "A2", note); err != nil {
		return err
	}
	if err := f.MergeCell(sheetSummary, "A2", "D2"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetSummary, "A2", "D2", s.note); err != nil {
		return err
	}
	if err := f.SetRowHeight(sheetSummary, 2, 62); err != nil {
		return err
	}

	if err := f.SetCellStr(sheetSummary, "A4", "Metric"); err != nil {
		return err
	}
	if err := f.SetCellStr(sheetSummary, "B4", "Value"); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetSummary, "A4", "B4", s.header); err != nil {
		return err
	}
	if err := f.SetCellStr(sheetSummary, "C4", "Row 5 is a typed count. Rows 6 through 11 are formulas. They read the other sheets."); err != nil {
		return err
	}
	if err := f.SetCellStyle(sheetSummary, "C4", "C4", s.note); err != nil {
		return err
	}

	cleanLast := len(res.Clean) + 1
	exceptionLast := len(res.Exceptions) + 1
	metrics := []struct {
		row   int
		label string
		value string
		typed bool
	}{
		{5, "Rows submitted", "", true},
		{6, "Clean rows", aggFormula("COUNTA", sheetClean, "A", "", cleanLast, len(res.Clean)), false},
		{7, "Full-time months", aggFormula("COUNTIF", sheetClean, "G", "TRUE", cleanLast, len(res.Clean)), false},
		{8, "Part-time months", aggFormula("COUNTIF", sheetClean, "G", "FALSE", cleanLast, len(res.Clean)), false},
		{9, "Coverage gaps", aggFormula("COUNTIF", sheetClean, "H", "TRUE", cleanLast, len(res.Clean)), false},
		{10, "Total hours", aggFormula("SUM", sheetClean, "E", "", cleanLast, len(res.Clean)), false},
		{11, "Exception items", aggFormula("COUNTA", sheetExceptions, "A", "", exceptionLast, len(res.Exceptions)), false},
	}
	for _, metric := range metrics {
		label := fmt.Sprintf("A%d", metric.row)
		value := fmt.Sprintf("B%d", metric.row)
		if err := f.SetCellStr(sheetSummary, label, metric.label); err != nil {
			return err
		}
		if metric.typed {
			if err := f.SetCellInt(sheetSummary, value, int64(res.Summary.InputRows)); err != nil {
				return err
			}
			if err := f.SetCellStyle(sheetSummary, value, value, s.typed); err != nil {
				return err
			}
			continue
		}
		if err := f.SetCellFormula(sheetSummary, value, metric.value); err != nil {
			return err
		}
	}
	if err := f.SetColWidth(sheetSummary, "A", "A", 24); err != nil {
		return err
	}
	if err := f.SetColWidth(sheetSummary, "B", "B", 16); err != nil {
		return err
	}
	if err := f.SetColWidth(sheetSummary, "C", "C", 78); err != nil {
		return err
	}
	return writeMonthChart(f, res.Clean, cleanLast)
}

func aggFormula(fn, sheet, col, criteria string, last, n int) string {
	if n == 0 {
		return "0"
	}
	quoted := sheet
	if sheet == sheetClean {
		quoted = "'" + sheet + "'"
	}
	ref := fmt.Sprintf("%s!$%s$2:$%s$%d", quoted, col, col, last)
	if criteria == "" {
		return fmt.Sprintf("%s(%s)", fn, ref)
	}
	return fmt.Sprintf("%s(%s,%s)", fn, ref, criteria)
}

func writeMonthChart(f *excelize.File, rows []record.Row, cleanLast int) error {
	if err := f.SetCellStr(sheetSummary, "A13", "Hours by month"); err != nil {
		return err
	}
	if len(rows) == 0 {
		return f.SetCellStr(sheetSummary, "A14", "No clean rows to chart.")
	}
	for i, name := range []string{"Month", "Hours", "Full-time months"} {
		cell, err := excelize.CoordinatesToCellName(i+1, 14)
		if err != nil {
			return err
		}
		if err := f.SetCellStr(sheetSummary, cell, name); err != nil {
			return err
		}
	}
	seen := map[string]bool{}
	var months []string
	for _, row := range rows {
		if !seen[row.Month] {
			seen[row.Month] = true
			months = append(months, row.Month)
		}
	}
	slices.Sort(months)
	for i, month := range months {
		excelRow := 15 + i
		if err := f.SetCellStr(sheetSummary, fmt.Sprintf("A%d", excelRow), SafeText(month)); err != nil {
			return err
		}
		hours := fmt.Sprintf("SUMIF('Clean Data'!$D$2:$D$%d,A%d,'Clean Data'!$E$2:$E$%d)", cleanLast, excelRow, cleanLast)
		full := fmt.Sprintf("COUNTIFS('Clean Data'!$D$2:$D$%d,A%d,'Clean Data'!$G$2:$G$%d,TRUE)", cleanLast, excelRow, cleanLast)
		if err := f.SetCellFormula(sheetSummary, fmt.Sprintf("B%d", excelRow), hours); err != nil {
			return err
		}
		if err := f.SetCellFormula(sheetSummary, fmt.Sprintf("C%d", excelRow), full); err != nil {
			return err
		}
	}
	end := 14 + len(months)
	return f.AddChart(sheetSummary, "E4", &excelize.Chart{
		Type: excelize.Col,
		Series: []excelize.ChartSeries{{
			Name:       "Summary!$B$14",
			Categories: fmt.Sprintf("Summary!$A$15:$A$%d", end),
			Values:     fmt.Sprintf("Summary!$B$15:$B$%d", end),
		}},
		Title:  []excelize.RichTextRun{{Text: "Hours by month"}},
		Legend: excelize.ChartLegend{Position: "bottom"},
		YAxis: excelize.ChartAxis{
			Title: []excelize.RichTextRun{{Text: "Hours"}},
		},
		Format: excelize.GraphicOptions{ScaleX: 1.15, ScaleY: 1.15},
	})
}

func writePivot(f *excelize.File, rows []record.Row) error {
	if err := f.SetCellStr(sheetPivot, "A1", "Hours by month. Summary is the sheet whose counts are formulas."); err != nil {
		return err
	}
	if len(rows) == 0 {
		return f.SetCellStr(sheetPivot, "A3", "No clean rows, so no pivot table was created.")
	}
	return f.AddPivotTable(&excelize.PivotTableOptions{
		DataRange:       fmt.Sprintf("Clean Data!A1:E%d", len(rows)+1),
		PivotTableRange: "Pivot!A3:C20",
		Rows:            []excelize.PivotTableField{{Data: "Month"}},
		Data: []excelize.PivotTableField{{
			Data: "Hours worked", Name: "Total hours", Subtotal: "Sum",
		}},
		RowGrandTotals:      true,
		ColGrandTotals:      true,
		ShowRowHeaders:      true,
		ShowColHeaders:      true,
		UseAutoFormatting:   true,
		PivotTableStyleName: "PivotStyleLight16",
	})
}
