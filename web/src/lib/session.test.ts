import { describe, expect, it } from "vitest";
import { groupExceptions, movementMessage, uploadMessage, workingSet } from "./session";
import type { AuditResult, CleanRow, Exception, Summary } from "./types";

const summary: Summary = {
  input_rows: 2,
  clean_rows: 1,
  exception_count: 1,
  rows_with_errors: 1,
  full_time_months: 1,
  part_time_months: 0,
  coverage_gaps: 0,
  total_hours: 160,
};

function clean(row: number): CleanRow {
  return {
    row,
    employee_id: "E001",
    name: "Ada",
    hire_date: "2020-01-15",
    month: "2024-01",
    hours_worked: 160,
    coverage_offered: true,
    full_time: true,
    coverage_gap: false,
  };
}

function problem(row: number, field: string, reason: string): Exception {
  return {
    row,
    field,
    reason,
    raw: {
      employee_id: "E002",
      name: "Grace",
      hire_date: "nope",
      month: "2024-01",
      hours_worked: "10",
      coverage_offered: "maybe",
    },
  };
}

describe("groupExceptions", () => {
  it("keeps one editable row when a source row has two problems", () => {
    const groups = groupExceptions([
      problem(4, "hire_date", "bad date"),
      problem(4, "coverage_offered", "not yes or no"),
      problem(9, "hours_worked", "not a number"),
    ]);
    expect(groups).toHaveLength(2);
    expect(groups[0]?.issues.map((issue) => issue.field)).toEqual(["hire_date", "coverage_offered"]);
    expect(groups[0]?.raw.employee_id).toBe("E002");
    expect(groups[1]?.row).toBe(9);
  });
});

describe("workingSet", () => {
  it("sends each source row once, clean rows included", () => {
    const review = groupExceptions([problem(4, "hire_date", "bad date"), problem(4, "coverage_offered", "no")]);
    review[0]!.raw.hire_date = "2020-01-01";
    const rows = workingSet([clean(2)], review);
    expect(rows.map((row) => row.row)).toEqual([2, 4]);
    expect(rows[0]).toMatchObject({ hours_worked: "160", coverage_offered: "true" });
    expect(rows[1]?.hire_date).toBe("2020-01-01");
  });
});

describe("messages", () => {
  it("describes the first upload", () => {
    const result: AuditResult = { clean: [clean(2)], exceptions: [problem(4, "hire_date", "bad")], summary, warnings: [] };
    expect(uploadMessage(result)).toBe("2 rows submitted. 1 clean. 1 in review.");
  });

  it("names the row that moved and the row that stayed", () => {
    const next: AuditResult = {
      clean: [clean(2), { ...clean(4), employee_id: "E002" }],
      exceptions: [problem(9, "hours_worked", "not a number")],
      summary,
      warnings: [],
    };
    expect(movementMessage([problem(4, "hire_date", "bad"), problem(9, "hours_worked", "no")], next)).toBe(
      "Row 4 moved to clean data. row 9 is still in review.",
    );
  });

  it("says when a recheck moved nothing", () => {
    const next: AuditResult = { clean: [], exceptions: [problem(4, "hire_date", "bad")], summary, warnings: [] };
    expect(movementMessage([problem(4, "hire_date", "bad")], next)).toBe(
      "No rows moved to clean data. row 4 is still in review.",
    );
  });

  it("does not treat a vanished row as deleted", () => {
    const next: AuditResult = { clean: [], exceptions: [], summary, warnings: [] };
    expect(movementMessage([problem(4, "hire_date", "bad")], next)).toContain("not removed here");
  });
});
