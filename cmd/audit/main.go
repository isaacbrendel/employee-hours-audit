package main

import (
	"bytes"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/isaacbrendel/employee-hours-audit/internal/record"
	"github.com/isaacbrendel/employee-hours-audit/internal/report"
)

func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

func run(args []string, stdout, stderr io.Writer) int {
	fs := flag.NewFlagSet("audit", flag.ContinueOnError)
	fs.SetOutput(stderr)
	out := fs.String("o", "hours-audit.xlsx", "workbook to write")
	fs.Usage = func() {
		fmt.Fprintln(stderr, "usage: audit <file.csv> [-o report.xlsx]")
		fs.PrintDefaults()
	}
	if err := fs.Parse(normalizeArgs(args)); err != nil {
		return 2
	}
	if fs.NArg() != 1 {
		fs.Usage()
		return 2
	}

	body, err := os.ReadFile(fs.Arg(0))
	if err != nil {
		fmt.Fprintf(stderr, "read %s: %v\n", fs.Arg(0), err)
		return 1
	}
	res, err := record.AuditCSV(bytes.NewReader(body))
	if err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	var buf bytes.Buffer
	if err := report.Write(&buf, res); err != nil {
		fmt.Fprintln(stderr, err)
		return 1
	}
	if err := os.WriteFile(*out, buf.Bytes(), 0o644); err != nil {
		fmt.Fprintf(stderr, "write %s: %v\n", *out, err)
		return 1
	}
	for _, warning := range res.Warnings {
		fmt.Fprintln(stderr, "warning:", warning)
	}
	s := res.Summary
	fmt.Fprintf(stdout, "Rows submitted:   %d\n", s.InputRows)
	fmt.Fprintf(stdout, "Clean rows:       %d\n", s.CleanRows)
	fmt.Fprintf(stdout, "Rows with errors: %d\n", s.RowsWithErrors)
	fmt.Fprintf(stdout, "Exception items:  %d\n", s.ExceptionCount)
	fmt.Fprintf(stdout, "Full-time months: %d\n", s.FullTimeMonths)
	fmt.Fprintf(stdout, "Coverage gaps:    %d\n", s.CoverageGaps)
	fmt.Fprintf(stdout, "Total hours:      %g\n", s.TotalHours)
	fmt.Fprintf(stdout, "Wrote %s\n", *out)
	return 0
}

// normalizeArgs lets -o follow the filename. flag.FlagSet stops at the first
// non-flag, and the documented command is `audit file.csv -o report.xlsx`.
func normalizeArgs(args []string) []string {
	var flags, files []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch {
		case arg == "-o" || arg == "--o":
			flags = append(flags, "-o")
			if i+1 < len(args) {
				i++
				flags = append(flags, args[i])
			}
		case strings.HasPrefix(arg, "-o="):
			flags = append(flags, "-o", strings.TrimPrefix(arg, "-o="))
		case arg == "-h" || arg == "-help" || arg == "--help":
			flags = append(flags, "-h")
		default:
			files = append(files, arg)
		}
	}
	return append(flags, files...)
}
