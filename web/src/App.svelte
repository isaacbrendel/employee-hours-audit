<script lang="ts">
  import { onMount } from "svelte";
  import Review from "./lib/Review.svelte";
  import { downloadReport, health, suggestFix, validateFile, validateRows } from "./lib/api";
  import { groupExceptions, movementMessage, uploadMessage, workingSet } from "./lib/session";
  import type { AuditResult, Column, ReviewRow } from "./lib/types";
  import type { Suggestion } from "./lib/api";

  type Stage = "upload" | "review" | "export";

  let stage = $state<Stage>("upload");
  let result = $state<AuditResult | null>(null);
  let review = $state<ReviewRow[]>([]);
  let message = $state("");
  let error = $state("");
  let busy = $state(false);
  let suggestEnabled = $state(false);
  let chosen = $state<File | null>(null);
  let asking = $state("");
  let suggestion = $state<(Suggestion & { row: number }) | null>(null);

  onMount(() => {
    health()
      .then((status) => {
        suggestEnabled = status.suggest;
      })
      .catch(() => {
        suggestEnabled = false;
      });
  });

  function resetFrom(next: AuditResult, note: string) {
    result = next;
    review = groupExceptions(next.exceptions);
    message = note;
    suggestion = null;
    error = "";
  }

  async function checkFile() {
    if (!chosen) {
      error = "Choose a CSV file first.";
      return;
    }
    busy = true;
    error = "";
    try {
      const next = await validateFile(chosen);
      resetFrom(next, uploadMessage(next));
      stage = "review";
    } catch (err) {
      error = err instanceof Error ? err.message : "The file could not be checked.";
    } finally {
      busy = false;
    }
  }

  async function recheck() {
    if (!result) return;
    busy = true;
    error = "";
    const before = result.exceptions;
    try {
      const next = await validateRows(workingSet(result.clean, review));
      const note = movementMessage(before, next);
      resetFrom(next, note);
    } catch (err) {
      error = err instanceof Error ? err.message : "The rows could not be rechecked.";
    } finally {
      busy = false;
    }
  }

  async function exportWorkbook() {
    if (!result) return;
    busy = true;
    error = "";
    try {
      const blob = await downloadReport(workingSet(result.clean, review));
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = "hours-audit.xlsx";
      link.click();
      URL.revokeObjectURL(url);
      message = "Downloaded hours-audit.xlsx. Summary counts in that file are formulas.";
    } catch (err) {
      error = err instanceof Error ? err.message : "The workbook could not be downloaded.";
    } finally {
      busy = false;
    }
  }

  async function ask(row: ReviewRow, field: string, reason: string) {
    asking = `${row.row}:${field}`;
    error = "";
    try {
      const got = await suggestFix(field, reason, row.raw);
      suggestion = { ...got, row: row.row, field };
    } catch (err) {
      error = err instanceof Error ? err.message : "No suggestion was returned.";
    } finally {
      asking = "";
    }
  }

  function useSuggestion(rowNumber: number, field: Column, value: string) {
    const row = review.find((item) => item.row === rowNumber);
    if (row) {
      row.raw[field] = value;
    }
    suggestion = null;
    message = `Row ${rowNumber} now has “${value}” in ${field}. It stays in review until you recheck.`;
  }

  function startOver() {
    stage = "upload";
    result = null;
    review = [];
    chosen = null;
    message = "";
    error = "";
    suggestion = null;
  }
</script>

<div class="shell">
  <header>
    <div>
      <h1>Hours audit</h1>
      <p>A demo for messy monthly hours. Not tax, legal, or ACA filing advice.</p>
    </div>
  </header>

  <ol class="steps">
    <li><button type="button" aria-current={stage === "upload" ? "step" : undefined} onclick={() => (stage = "upload")}>1 Upload</button></li>
    <li><button type="button" aria-current={stage === "review" ? "step" : undefined} onclick={() => result && (stage = "review")} disabled={!result}>2 Review</button></li>
    <li><button type="button" aria-current={stage === "export" ? "step" : undefined} onclick={() => result && (stage = "export")} disabled={!result}>3 Export</button></li>
  </ol>

  {#if error}
    <p class="error" role="alert">{error}</p>
  {/if}
  {#if message}
    <p class="status" role="status">{message}</p>
  {/if}

  {#if stage === "upload"}
    <section class="panel">
      <h2>Check a file</h2>
      <p class="lede">Columns: employee id, name, hire date, month, hours, coverage offered. Bad rows stay in the result with a reason.</p>
      <label class="file">
        Hours CSV
        <input
          type="file"
          accept=".csv,text/csv"
          onchange={(event) => {
            const input = event.currentTarget as HTMLInputElement;
            chosen = input.files?.[0] ?? null;
          }}
        />
      </label>
      <div class="actions">
        <button type="button" onclick={checkFile} disabled={busy}>{busy ? "Checking…" : "Check this file"}</button>
      </div>
    </section>
  {/if}

  {#if result && stage === "review"}
    <section class="panel">
      <h2>What the check found</h2>
      <dl class="counts">
        <div><dt>Submitted</dt><dd>{result.summary.input_rows}</dd></div>
        <div><dt>Clean</dt><dd>{result.summary.clean_rows}</dd></div>
        <div><dt>In review</dt><dd>{result.summary.rows_with_errors}</dd></div>
        <div><dt>Full time</dt><dd>{result.summary.full_time_months}</dd></div>
        <div><dt>Coverage gaps</dt><dd>{result.summary.coverage_gaps}</dd></div>
      </dl>
      {#each result.warnings as warning (warning)}
        <p class="warning">{warning}</p>
      {/each}

      {#if review.length === 0}
        <p>Every row passed. Nothing is waiting for review.</p>
      {:else}
        <Review
          rows={review}
          {suggestEnabled}
          {asking}
          {suggestion}
          onSuggest={ask}
          onUse={useSuggestion}
          onDismiss={() => (suggestion = null)}
        />
      {/if}

      <div class="actions">
        <button type="button" onclick={recheck} disabled={busy || review.length === 0}>Recheck rows</button>
        <button type="button" class="secondary" onclick={() => (stage = "export")}>Continue to export</button>
      </div>

      <h2 class="subhead">Clean data</h2>
      <p class="lede">These rows passed. A full-time month with no coverage stays here and is marked as a gap. It is not dropped.</p>
      {#if result.clean.length === 0}
        <p>No clean rows yet.</p>
      {:else}
        <div class="scroll">
          <table>
            <caption class="lede">Rows that passed validation</caption>
            <thead>
              <tr>
                <th scope="col">Row</th>
                <th scope="col">Employee ID</th>
                <th scope="col">Name</th>
                <th scope="col">Month</th>
                <th scope="col">Hours</th>
                <th scope="col">Coverage</th>
              </tr>
            </thead>
            <tbody>
              {#each result.clean as row (row.row)}
                <tr>
                  <th scope="row">{row.row}</th>
                  <td>{row.employee_id}</td>
                  <td>{row.name}</td>
                  <td>{row.month}</td>
                  <td>{row.hours_worked}</td>
                  <td>
                    {row.coverage_offered ? "yes" : "no"}
                    {#if row.full_time}<span class="tag full">Full time</span>{/if}
                    {#if row.coverage_gap}<span class="tag gap">No coverage</span>{/if}
                  </td>
                </tr>
              {/each}
            </tbody>
          </table>
        </div>
      {/if}
    </section>
  {/if}

  {#if result && stage === "export"}
    <section class="panel">
      <h2>Workbook</h2>
      <p class="lede">Summary counts are Excel formulas that read Clean Data and Exceptions. Full time in the sheet is <code>=hours&gt;=130</code>, not a number pasted from this page.</p>
      <dl class="counts">
        <div><dt>Clean</dt><dd>{result.summary.clean_rows}</dd></div>
        <div><dt>Exception items</dt><dd>{result.summary.exception_count}</dd></div>
        <div><dt>Coverage gaps</dt><dd>{result.summary.coverage_gaps}</dd></div>
      </dl>
      <div class="actions">
        <button type="button" onclick={exportWorkbook} disabled={busy}>{busy ? "Building…" : "Download workbook"}</button>
        <button type="button" class="secondary" onclick={() => (stage = "review")}>Back to review</button>
        <button type="button" class="secondary" onclick={startOver}>Check another file</button>
      </div>
    </section>
  {/if}
</div>
