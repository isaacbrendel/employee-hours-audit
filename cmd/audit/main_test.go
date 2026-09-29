package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCLI(t *testing.T) {
	dir := t.TempDir()
	out := filepath.Join(dir, "report.xlsx")
	var stdout, stderr bytes.Buffer
	code := run([]string{"../../testdata/employees_messy.csv", "-o", out}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, stdout.String(), stderr.String())
	}
	if !strings.Contains(stdout.String(), "Rows submitted:   46") || !strings.Contains(stdout.String(), "Wrote "+out) {
		t.Fatalf("stdout:\n%s", stdout.String())
	}
	if !strings.Contains(stderr.String(), "Department") {
		t.Fatalf("stderr:\n%s", stderr.String())
	}
	info, err := os.Stat(out)
	if err != nil || info.Size() == 0 {
		t.Fatalf("workbook: %v", err)
	}

	stdout.Reset()
	stderr.Reset()
	outFirst := filepath.Join(dir, "first.xlsx")
	code = run([]string{"-o", outFirst, "../../testdata/employees_messy.csv"}, &stdout, &stderr)
	if code != 0 {
		t.Fatalf("flag-first exit %d %s", code, stderr.String())
	}
}

func TestCLIUsage(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run(nil, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "usage:") {
		t.Fatalf("exit %d stderr %s", code, stderr.String())
	}
}

func TestCLIMissingFile(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"does-not-exist.csv"}, &stdout, &stderr); code != 1 {
		t.Fatalf("exit %d", code)
	}
}

func TestCLIBadHeader(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.csv")
	if err := os.WriteFile(path, []byte("name,hours\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	var stdout, stderr bytes.Buffer
	if code := run([]string{path}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "missing required column") {
		t.Fatalf("exit %d stderr %s", code, stderr.String())
	}
}
