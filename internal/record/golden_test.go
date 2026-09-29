package record

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

func TestGolden(t *testing.T) {
	body, err := os.ReadFile("../../testdata/employees_messy.csv")
	if err != nil {
		t.Fatal(err)
	}
	got, err := AuditCSV(bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	encoded, err := json.MarshalIndent(got, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	encoded = append(encoded, '\n')

	path := "../../testdata/employees_messy.golden.json"
	if os.Getenv("UPDATE_GOLDEN") == "1" {
		if err := os.WriteFile(path, encoded, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	want, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(want, encoded) {
		t.Fatalf("golden mismatch (%d bytes vs %d). Set UPDATE_GOLDEN=1 only after reviewing the diff.", len(want), len(encoded))
	}
}
