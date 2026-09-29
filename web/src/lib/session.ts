import type { AuditResult, CleanRow, Exception, Input, ReviewRow } from "./types";

export function groupExceptions(items: Exception[]): ReviewRow[] {
  const order: number[] = [];
  const groups = new Map<number, ReviewRow>();
  for (const item of items) {
    let group = groups.get(item.row);
    if (!group) {
      group = { row: item.row, raw: { ...item.raw }, issues: [] };
      groups.set(item.row, group);
      order.push(item.row);
    }
    group.issues.push({ field: item.field, reason: item.reason });
  }
  return order.map((row) => {
    const group = groups.get(row);
    if (!group) {
      throw new Error(`missing review row ${row}`);
    }
    return group;
  });
}

export function formatHours(hours: number): string {
  if (!Number.isFinite(hours)) {
    return "";
  }
  return String(hours);
}

export function cleanToInput(row: CleanRow): Input {
  return {
    row: row.row,
    employee_id: row.employee_id,
    name: row.name,
    hire_date: row.hire_date,
    month: row.month,
    hours_worked: formatHours(row.hours_worked),
    coverage_offered: row.coverage_offered ? "true" : "false",
  };
}

export function workingSet(clean: CleanRow[], review: ReviewRow[]): Input[] {
  return [...clean.map(cleanToInput), ...review.map((row) => ({ row: row.row, ...row.raw }))].sort(
    (a, b) => a.row - b.row,
  );
}

export function uploadMessage(result: AuditResult): string {
  const summary = result.summary;
  return `${summary.input_rows} rows submitted. ${summary.clean_rows} clean. ${summary.rows_with_errors} in review.`;
}

export function movementMessage(before: Exception[], after: AuditResult): string {
  const beforeRows = [...new Set(before.map((item) => item.row))].sort((a, b) => a - b);
  if (beforeRows.length === 0) {
    return "There were no rows in review.";
  }
  const clean = new Set(after.clean.map((row) => row.row));
  const still = new Set(after.exceptions.map((item) => item.row));
  const moved = beforeRows.filter((row) => clean.has(row));
  const remaining = beforeRows.filter((row) => still.has(row));
  const gone = beforeRows.filter((row) => !clean.has(row) && !still.has(row));

  const sentences: string[] = [];
  if (moved.length === 0) {
    sentences.push("No rows moved to clean data");
  } else if (moved.length === 1) {
    sentences.push(`Row ${moved[0]} moved to clean data`);
  } else {
    sentences.push(`Rows ${join(moved)} moved to clean data`);
  }
  if (remaining.length === 1) {
    sentences.push(`row ${remaining[0]} is still in review`);
  } else if (remaining.length > 1) {
    sentences.push(`${remaining.length} rows are still in review`);
  }
  if (gone.length > 0) {
    sentences.push(`rows ${join(gone)} are missing from the result and were not removed here`);
  }
  return sentences.join(". ") + ".";
}

function join(rows: number[]): string {
  return rows.join(", ");
}
