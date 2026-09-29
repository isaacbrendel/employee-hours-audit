package record

import (
	"bytes"
	"strings"
	"testing"
)

func TestParseShapes(t *testing.T) {
	header := "employee_id,name,hire_date,month,hours_worked,coverage_offered"

	tests := []struct {
		name    string
		csv     string
		wantErr string
		check   func(t *testing.T, got Result)
	}{
		{
			name:    "empty",
			csv:     " \n\n",
			wantErr: "empty",
		},
		{
			name:    "not utf-8",
			csv:     "employee_id,name,hire_date,month,hours_worked,coverage_offered\nE001,A" + string([]byte{0xFF}) + ",2020-01-01,2024-01,10,yes\n",
			wantErr: "UTF-8",
		},
		{
			name:    "missing column",
			csv:     "employee_id,name,hire_date,month,hours_worked\n",
			wantErr: "coverage_offered",
		},
		{
			name:    "duplicate header",
			csv:     "employee_id,name,hire_date,month,hours_worked,coverage_offered,Name\n",
			wantErr: "appears twice",
		},
		{
			name:    "empty header name",
			csv:     "employee_id,name,hire_date,month,hours_worked,coverage_offered,\n",
			wantErr: "empty column",
		},
		{
			name:    "broken quote stops the file",
			csv:     header + "\nE001,\"Ada,2020-01-01,2024-01,10,yes\n",
			wantErr: "line",
		},
		{
			name: "header only",
			csv:  header + "\n",
			check: func(t *testing.T, got Result) {
				if got.Summary.InputRows != 0 || len(got.Clean) != 0 || len(got.Warnings) != 0 {
					t.Fatalf("%+v", got)
				}
			},
		},
		{
			name: "bom and mixed header case",
			csv:  "\uFEFFEmployee ID,NAME,Hire Date,Month,Hours Worked,Coverage Offered\nE001,Ada,2020-01-15,2024-01,10,yes\n",
			check: func(t *testing.T, got Result) {
				if len(got.Clean) != 1 || got.Clean[0].Name != "Ada" || got.Clean[0].Row != 2 {
					t.Fatalf("%+v", got)
				}
			},
		},
		{
			name: "crlf quoted comma and trailing blank lines",
			csv:  header + "\r\nE003,\"Alan Turing, Jr.\",2019-03-01,2024-01,80,N\r\n\r\n",
			check: func(t *testing.T, got Result) {
				if len(got.Clean) != 1 || got.Clean[0].Name != "Alan Turing, Jr." || got.Clean[0].CoverageOffered {
					t.Fatalf("%+v", got)
				}
			},
		},
		{
			name: "short row is not mapped onto the next record",
			csv:  header + "\nE040,Short Row,2020-01-01,2024-10,100\nE001,Ada,2020-01-15,2024-01,10,yes\n",
			check: func(t *testing.T, got Result) {
				if len(got.Clean) != 1 || got.Clean[0].EmployeeID != "E001" {
					t.Fatalf("clean %+v", got.Clean)
				}
				if len(got.Exceptions) != 1 || got.Exceptions[0].Field != "row" || got.Exceptions[0].Row != 2 {
					t.Fatalf("%+v", got.Exceptions)
				}
				if !strings.Contains(got.Exceptions[0].Reason, "5 columns") {
					t.Fatalf("%s", got.Exceptions[0].Reason)
				}
			},
		},
		{
			name: "blank line in the middle does not consume a data row",
			csv:  header + "\nE001,Ada,2020-01-15,2024-01,10,yes\n\nE002,Grace,2018-06-01,2024-01,10,yes\n",
			check: func(t *testing.T, got Result) {
				if len(got.Clean) != 2 || got.Clean[1].Row != 4 {
					t.Fatalf("%+v", got.Clean)
				}
			},
		},
		{
			name: "extra column is a warning",
			csv:  header + ",Department\nE001,Ada,2020-01-15,2024-01,10,yes,Engineering\n",
			check: func(t *testing.T, got Result) {
				if len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "Department") {
					t.Fatalf("%v", got.Warnings)
				}
				if len(got.Clean) != 1 {
					t.Fatalf("%+v", got)
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := AuditCSV(strings.NewReader(tt.csv))
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("err=%v", err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			tt.check(t, got)
		})
	}
}

func TestParseBytesBOM(t *testing.T) {
	body := append([]byte{0xEF, 0xBB, 0xBF}, []byte("employee_id,name,hire_date,month,hours_worked,coverage_offered\nE001,Ada,2020-01-15,2024-01,10,yes\n")...)
	got, err := AuditCSV(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Clean) != 1 {
		t.Fatalf("%+v", got)
	}
}
