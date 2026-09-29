package record

import (
	"bytes"
	"os"
	"testing"
)

func FuzzParse(f *testing.F) {
	f.Add([]byte(""))
	f.Add([]byte("employee_id,name,hire_date,month,hours_worked,coverage_offered\n"))
	f.Add([]byte("\xef\xbb\xbfEmployee ID,NAME,Hire Date,Month,Hours Worked,Coverage Offered\r\nE001,\"Ada, Jr.\",03/15/2015,2024-01,160,yes\r\n\r\n"))
	f.Add([]byte("employee_id,name,hire_date,month,hours_worked,coverage_offered\nE001,\"unterminated,2020-01-01,2024-01,10,yes\n"))
	if body, err := os.ReadFile("../../testdata/employees_messy.csv"); err == nil {
		f.Add(body)
	}
	f.Fuzz(func(t *testing.T, data []byte) {
		_, _ = AuditCSV(bytes.NewReader(data))
	})
}
