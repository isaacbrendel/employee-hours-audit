import type { AuditResult, Input, Raw } from "./types";

export interface Suggestion {
  field: string;
  suggested: string;
  explanation: string;
}

export async function health(): Promise<{ status: string; suggest: boolean }> {
  const response = await fetch("/healthz");
  return readJSON(response);
}

export async function validateFile(file: File): Promise<AuditResult> {
  const body = new FormData();
  body.append("file", file);
  const response = await fetch("/api/validate", { method: "POST", body });
  return readJSON(response);
}

export async function validateRows(rows: Input[]): Promise<AuditResult> {
  const response = await fetch("/api/validate", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rows }),
  });
  return readJSON(response);
}

export async function downloadReport(rows: Input[]): Promise<Blob> {
  const response = await fetch("/api/report", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ rows }),
  });
  if (!response.ok) {
    const payload = await response.json().catch(() => ({ error: "could not build the workbook" }));
    throw new Error(payload.error ?? "could not build the workbook");
  }
  return response.blob();
}

export async function suggestFix(field: string, reason: string, raw: Raw): Promise<Suggestion> {
  const response = await fetch("/api/suggest", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ field, reason, raw }),
  });
  return readJSON(response);
}

async function readJSON<T>(response: Response): Promise<T> {
  const payload = await response.json().catch(() => ({ error: response.statusText }));
  if (!response.ok) {
    throw new Error(payload.error ?? "request failed");
  }
  return payload as T;
}
