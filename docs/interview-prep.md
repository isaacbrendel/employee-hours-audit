# Interview prep: coding project + AI verification

Talking points for Blake’s Thursday call. Scope: **(1) how the coding project was completed** and **(5) AI usage and verification**. Excel interview and TradeSiteUsa are out of this repo.

Live demo: https://employee-hours-audit.fly.dev/  
Fixture: `testdata/employees_messy.csv` → 46 submitted, 18 clean, 28 in review.

---

## 1. How I completed the coding project

### One-sentence pitch

I built Option C (covers B and D): a Go CLI + `net/http` API that audits messy monthly-hours CSVs, a Svelte review UI to fix bad rows, and an Excel workbook whose summary counts are formulas—not pasted numbers.

### Story arc (2–3 minutes)

1. **Problem.** Payroll-style CSVs arrive messy. A compliance reviewer needs every bad row kept with a reason, not silently dropped, and a workbook they can defend.
2. **Research before code.** IRS monthly measurement (130 hours/month), RFC 4180 / `encoding/csv`, OWASP upload + CSV-injection guidance, excelize for xlsx. Decisions were written down first (those notes were later removed from the public README so the demo stayed short).
3. **Vertical slice.** Parse → validate → CLI workbook → HTTP API → Svelte review/export → optional Anthropic suggest (human must accept) → deploy on Fly.
4. **Prove it.** Table tests, golden JSON for the planted fixture, race + coverage, 60s fuzz (~1.9M execs), Vitest, then a recorded headless Chrome pass through upload → fix row 45 → download.

### Design decisions worth saying out loud

| Choice | Why |
| --- | --- |
| Hold **every** duplicate `employee_id` + `month` | Keeping “the first” is a guess. Reviewer decides. |
| Coverage gap is a **flag on a clean row**, not an exception | Valid `no` is data; dropping it hides the case that matters. |
| Full-time = **≥130 hours**, monthly method only | IRS citing; look-back / 1095-C codes / affordability intentionally out of scope. |
| Slash dates = **MDY**; `13/01/2024` fails | US reporting demo; no silent DMY reinterpretation. |
| Hours: plain decimal, 0–744; no commas / sci-notation | Guessing `1,200` or capping `800` invents data. |
| Summary metrics are **Excel formulas** | Reviewer can audit the sheet; only “rows submitted” is typed (exception lines ≠ source rows). |
| `SafeText` prefixes formula triggers (`=`, `+`, `-`, `@`, …) | OWASP CSV injection; write xlsx cells, not CSV. |
| Stateless API, 1 MiB, `.csv` allowlist | Uploads parsed and discarded; no DB/login for this demo. |
| Go 1.22 ServeMux patterns; UI on `/` and `/assets/` only | So a catch-all file server cannot swallow `/api/*`. |

### Architecture in one breath

```
CSV/JSON  →  internal/record (parse + validate)
                ↓
         Result{Clean, Exceptions, Summary}
                ↓
    ┌───────────┴───────────┐
 CLI workbook          HTTP API
 (cmd/audit)     validate / report / suggest
                         ↓
                   Svelte UI (browser holds working set)
                         ↓
                   report.Write → .xlsx (Summary, Clean, Exceptions, Pivot)
```

Packages stay small: `record` owns rules, `report` owns excelize, `api` owns HTTP, `suggest` is optional and never mutates a row.

### What “done” looked like

- CLI exit codes: 0 = workbook written (even with review rows), 1 = read/parse failure, 2 = usage.
- Fixture behavior locked in `employees_messy.golden.json`.
- Manual path: fix E044 hours `nope` / coverage `perhaps` → `40` / `yes` → recheck moves row 45 to clean → download workbook with `COUNTIF` formulas and formula-safe names like `'=1+1`.
- Deployed demo on Fly; README points at it.

### Likely follow-ups

**Why not look-back?** Columns aren’t there; a wrong “official” answer is worse than an honest scope cut.

**Why not auto-resolve duplicates?** Same reason as AI suggest: the tool surfaces conflict; a person decides.

**Why formulas instead of computed cells?** Assignment asked for Excel reporting reviewers can trust; formulas recompute when the sheet changes and are inspectable.

**What would you add next?** Auth, look-back measurement, persistence of review sessions, real antivirus/scanning—still keeping validation deterministic and human-gated.

---

## 5. How I use AI, and how I verify the output

### Stance (say this first)

An agent wrote most of the scaffolding. I treated it as a fast junior: useful for boilerplate and first drafts, **not** trusted for product judgment or silent data changes. Verification is tests + fixtures + a real browser pass, not “it compiles.”

### Process I actually used

1. **I set the rules.** Validation policy (duplicates held, coverage gap as flag, MDY dates, hour bounds, formula injection) came from research—not from the model inventing ACA logic.
2. **Agent implements under those rules.** Packages, tests, UI flow, excelize wiring.
3. **I review diffs against the rules.** Especially parser edge cases, HTTP routing, Excel formula strings, and anything that mutates user data.
4. **Automated gates.** `go vet`, `go test -race -cover`, fuzz parse, Vitest, golden fixture.
5. **Black-box pass.** Upload the planted CSV, edit a known bad row, recheck, open the xlsx and confirm sheets / formulas / safe text.
6. **Document failures.** README and `docs/how-i-tested.md` name where the agent was wrong.

### Concrete agent mistakes I caught (good stories)

| Bug | Symptom | Fix / proof |
| --- | --- | --- |
| Flag parser ignored `-o` after the filename | `audit file.csv -o out.xlsx` did not match Go’s `flag` “stop at first non-flag” behavior | `normalizeArgs` reorders; CLI tests cover both orders |
| Pivot range quoted the sheet name wrongly | Broken / useless pivot | Correct range quoting; workbook tests assert a pivot exists |
| Catch-all `GET /` served API paths | `/api/...` could hit the static handler | Mount UI only on `/` and `/assets/`; method/path tests |

### Hard lines I would not let AI cross

- Choose which duplicate row to keep.
- Apply a suggested fix without the person clicking “use value” and rechecking.
- Invent look-back / penalty / 1095-C answers from incomplete columns.
- Skip tests that lock the planted fixture.

The suggest feature encodes that boundary: key off → 404; key on → returns a value; **server never writes it into the row** (`TestSuggestDoesNotApplyTheValue`). The UI only fills the field when the person accepts, then they must recheck.

### How I verify LLM / agent code (checklist you can recite)

1. **Does it match the written decision?** (e.g. duplicates held → assert both E037 rows in exceptions.)
2. **Table + golden tests** for the messy fixture and rule matrix.
3. **Fuzz** the parser so weird CSV bytes don’t panic.
4. **Race detector** on concurrent-sensitive paths.
5. **Integration:** real multipart upload, oversized body → 413, wrong type → 415.
6. **Open the artifact:** unzip xlsx, confirm formula strings and `SafeText` on `=1+1`.
7. **UI path:** fix → recheck → counts move; message like “Row 45 moved to clean data.”
8. **Negative tests for AI features:** suggest off; suggest does not mutate.

### Short answer if asked “Do you trust AI-generated code?”

No. I trust **tests and fixtures** that encode policy. AI is acceleration for typing and exploring APIs; the product rules and the verification harness are mine.

---

## Quick demo script (if they ask to walk through it)

1. Open https://employee-hours-audit.fly.dev/
2. Upload `testdata/employees_messy.csv` → expect 46 / 18 / 28 and the Department warning.
3. Find row 45 (E044, hours `nope`) → set hours `40`, coverage `yes` → Recheck → clean 19 / review 27.
4. Export → open Summary → point at formula cells and the disclaimer that this is not a 1094-C/1095-C.
5. Optionally: mention Suggest stays off without `ANTHROPIC_API_KEY`, and even when on it never auto-applies.

---

## What this doc does not cover

- Excel interview walkthrough (separate prep).
- TradeSiteUsa design decisions (separate repo / notes).
- General architecture drill beyond this project—use the package diagram above as the concrete example, then generalize (boundaries, invariants, human-in-the-loop for irreversible or judgment calls).
