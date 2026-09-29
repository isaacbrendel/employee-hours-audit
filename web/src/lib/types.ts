export interface Raw {
  employee_id: string;
  name: string;
  hire_date: string;
  month: string;
  hours_worked: string;
  coverage_offered: string;
}

export interface Input extends Raw {
  row: number;
}

export interface CleanRow {
  row: number;
  employee_id: string;
  name: string;
  hire_date: string;
  month: string;
  hours_worked: number;
  coverage_offered: boolean;
  full_time: boolean;
  coverage_gap: boolean;
}

export interface Exception {
  row: number;
  field: string;
  reason: string;
  raw: Raw;
}

export interface Summary {
  input_rows: number;
  clean_rows: number;
  exception_count: number;
  rows_with_errors: number;
  full_time_months: number;
  part_time_months: number;
  coverage_gaps: number;
  total_hours: number;
}

export interface AuditResult {
  clean: CleanRow[];
  exceptions: Exception[];
  summary: Summary;
  warnings: string[];
}

export interface Issue {
  field: string;
  reason: string;
}

export interface ReviewRow {
  row: number;
  raw: Raw;
  issues: Issue[];
}

export const columns = [
  "employee_id",
  "name",
  "hire_date",
  "month",
  "hours_worked",
  "coverage_offered",
] as const;

export type Column = (typeof columns)[number];

export const columnLabel: Record<Column, string> = {
  employee_id: "Employee ID",
  name: "Name",
  hire_date: "Hire date",
  month: "Month",
  hours_worked: "Hours",
  coverage_offered: "Coverage",
};
