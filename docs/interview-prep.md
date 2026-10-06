# Interview prep: talk for 15+ minutes (plain English)

This is a spoken outline for Blake’s call. Read it out loud once. Aim for **~15 minutes** on the coding project, then let AI verification take another few minutes if he asks, or fold it in as you go.

Live demo: https://employee-hours-audit.fly.dev/  
Sample file: `testdata/employees_messy.csv`

---

# Part A — The coding project (about 12–15 minutes)

## A1. Open with what it is (about 1 minute)

**Say something like:**

> I built a small employee-hours audit tool for messy monthly payroll CSVs.
>
> You upload a CSV of hours and coverage. The tool does not silently fix or drop bad rows. It splits the file into two buckets: rows a person can trust, and rows a person has to review, each with a plain-English reason.
>
> From there you can edit the bad rows in the browser, recheck them through the same rules, and download an Excel workbook. The important counts in that workbook are Excel formulas that read the data sheets—not numbers I pasted in from the server.
>
> It’s a demo for ACAPrime-style data cleanup work. It is not tax advice and it does not file Form 1094-C or 1095-C.

**If they ask “which option?”:** Option C on the brief—Go API + Svelte UI—also covering the CSV cleanup (B) and Excel reporting (D).

---

## A2. Why this problem exists (about 2 minutes)

**Plain English business context:**

ACAPrime / ChannelBound helps employers with Affordable Care Act reporting. In the real world, hours data often comes out of payroll systems as spreadsheets and CSVs that are ugly: wrong dates, blank IDs, “maybe” instead of yes/no, two rows for the same person in the same month, names that look like Excel formulas, extra columns nobody asked for.

Someone has to clean that before anything official happens. If software “helpfully” guesses—keeps the first duplicate, caps 800 hours down to 744, rewrites `13/01/2024` as January 13—you can hide the exact rows a human reviewer needed to see.

So the product pitch is: **parse honestly, hold everything unclear, let a person decide, then export something auditable.**

**The one ACA fact the tool actually uses:**

The IRS says that under the *monthly measurement method*, someone with at least **130 hours of service in a calendar month** counts as full-time for that month (30 hours/week × ~4⅓ weeks).

I implemented that flag only. I did **not** implement:

- look-back measurement (measure hours in an earlier period, freeze status later)
- affordability tests
- Form 1095-C line codes
- penalty math under 4980H

**Why leave those out?** The input columns don’t support them. Making up a “looks official” answer would be worse than saying “out of scope.” A full-time month with coverage = no is flagged as a *coverage gap* for the reviewer—it is not a determination that the employer owes a penalty.

---

## A3. Walk the life of one file (about 3–4 minutes)

Talk through this as a story. Use the planted fixture numbers: **46 rows in, 18 clean, 28 in review**.

### Step 1 — Upload / CLI read

Same core logic for CLI and web.

1. Read the whole file into memory (demo-sized; also capped).
2. Strip a UTF-8 BOM if present (Excel loves writing those).
3. Reject non-UTF-8 entirely—better a hard error than mojibake that looks “almost right.”
4. Use Go’s `encoding/csv`, not a hand-rolled split on commas. Quoted commas in names, CRLF line endings, blank lines—those are real.

**Header mapping in plain English:**  
Column names are trimmed, lowercased, spaces become underscores. So `Employee_ID`, `NAME`, and `Hours Worked` all map. Missing required columns fail the whole file. An extra column like `Department` becomes a **warning**, not a crash—and that column is not copied into clean data.

**Row numbers:** Physical line numbers. Header is line 1, first data row is line 2. Blank lines are not records, so trailing blank lines don’t invent fake exceptions.

**Broken rows:**

- Wrong column count → hold that row; do **not** slide fields left/right into the next person’s data.
- Broken quote (unclosed `"`) → stop the file. Continuing would invent garbage rows from misaligned bytes.

### Step 2 — Validate each row

Each field has a rule. Failures become **exceptions**: `{ row, field, reason, raw values }`. One row can have several exceptions (e.g. bad hours *and* bad coverage) so the reviewer can fix everything in one pass.

Examples you can name from the fixture:

| What showed up | What we do |
| --- | --- |
| Empty employee id | Hold: id required |
| Hire date `March 5th 2024` | Hold: not in the accepted date list |
| `13/01/2024` | Hold: slash dates are month/day/year; we do **not** reread as 13 January |
| Hours `abc`, `-4`, `1,200`, `1e2`, `800` | Hold; we don’t guess or cap |
| Coverage `maybe` / `offered` | Hold; only yes/no-style tokens |
| Hired in October for a September month | Hold: hire after reported month |
| Name `=1+1` | Allowed as data—but Excel export must store it as text, not a formula |
| Two rows for same employee + month | **Both** held; we don’t pick a winner |

**Clean rows** get normalized: hire date to `YYYY-MM-DD`, hours to a number, coverage to true/false, plus two derived flags:

- `full_time` = hours ≥ 130  
- `coverage_gap` = full-time AND coverage false  

Again: coverage gap stays on the clean sheet as a flag. A truthful “no” is information.

### Step 3 — Review in the browser

Three stages: Upload → Review → Export.

- Review groups exceptions by row so you see one person/row with a list of problems.
- You edit the raw fields.
- **Recheck** sends the whole working set (clean rows + edited review rows) back through the **same** Go validation. The browser does not reimplement the rules.
- Status text explains movement: e.g. “Row 45 moved to clean data. 27 rows are still in review.”

Optional: if `ANTHROPIC_API_KEY` is set, a “Suggest a fix” button can propose a value. Using it only fills the input. The row stays in review until you recheck. No key → suggest is hidden / 404.

### Step 4 — Excel workbook

Sheets: **Summary**, **Clean Data**, **Exceptions**, **Pivot**.

- Clean and Exceptions are the data.
- Summary counts (clean rows, full-time months, coverage gaps, total hours, exception items) are **formulas** like `COUNTIF` / `SUM` against those sheets.
- “Rows submitted” is the one typed number, because source rows and exception *items* are not 1:1 (one row can produce multiple exception lines; blank lines aren’t rows).
- Full-time and coverage-gap columns on Clean Data are formulas too (`=E2>=130`, etc.), so Excel recomputes them.
- Any text that starts with `=`, `+`, `-`, `@`, tab, etc. gets a leading `'` so spreadsheet apps treat it as text (OWASP CSV-injection idea, applied to xlsx cells).
- Pivot of hours by month is a convenience; the formulas are what I’d defend in review.
- Big disclaimer on Summary: demo only, not filing advice.

CLI does the same write path: `audit file.csv -o report.xlsx`, exit 0 even when some rows need review.

---

## A4. Design decisions — defend them like a conversation (about 4–5 minutes)

Don’t list them like a README. Tell them as **judgment calls**.

### “I refuse to guess”

**Duplicates:** If E037 appears twice for 2024-09, both rows go to review. Keeping the first is a silent policy decision the employer didn’t authorize. The reason string literally says a person should choose which record to keep.

**Dates:** Five accepted layouts only. Slash dates are US MDY. `02/31/2024` is rejected (we require the value to round-trip through the layout that parsed it—so Go can’t quietly roll it to March). Ambiguous international dates are failures, not cleverness.

**Hours:** Must look like a plain decimal (`160`, `37.5`). Thousands separators and scientific notation fail. Above 744 fails—we don’t cap. Zero hours is valid (reported zero ≠ missing).

**Coverage:** Small synonym set (yes/no, true/false, y/n, 1/0). “Offered” and “maybe” fail on purpose—they sound related but aren’t booleans.

### “Bad data stays visible”

Coverage gap on a clean full-time row with `no` is the case a compliance person cares about. If I rejected that row as invalid, I’d hide it. If I auto-flipped coverage to yes, I’d lie.

### “The workbook has to be inspectable”

Pasting summary numbers from the API would work until someone edits the sheet. Formulas mean the Summary tab is an audit trail against Clean Data / Exceptions. Tests check the formula *strings* and ranges; excelize does not run Excel’s calculation engine, and I’m honest about that.

### “Security for a demo, stated clearly”

- 1 MiB upload limit; larger → 413 (reject, don’t truncate).
- Must look like a `.csv`.
- Content-Type checked, but we still parse the bytes.
- Nothing stored; browser holds the working set.
- No logins—OWASP would want auth; README says we skipped it for scope.
- Formula-injection prefixing on export.

### “Small packages, one job each”

| Piece | Job |
| --- | --- |
| `internal/record` | Parse CSV + validate; never writes Excel or HTTP |
| `internal/report` | Turn a Result into xlsx |
| `internal/api` | HTTP only; calls record/report/suggest |
| `internal/suggest` | Optional LLM; returns a suggestion, mutates nothing |
| `cmd/audit` | CLI |
| `cmd/server` | Process wiring + static UI |
| `web/` | Svelte UI; proxies to API in dev |

That’s how I’d talk about architecture without buzzwords: **boundaries so rules live in one place**, and the UI can’t accidentally invent a different definition of full-time.

### “What I skipped and why”

Look-back, 1095-C codes, DB, accounts, embedding the UI in the binary (`go:embed` can’t reach `../web/dist` cleanly with gitignored build output)—all deliberate. The assignment asked for a review queue and an export, kept stateless.

---

## A5. How I proved it works (about 2 minutes)

**Automated**

- Table-driven tests for parse shapes and every validation rule.
- Golden JSON for `employees_messy.csv`—regenerate only after reading the diff.
- Duplicate tests (including identical duplicates still held).
- `go test -race`, coverage (~94% on `record`, ~80% module).
- Fuzz the parser 60 seconds (~1.9M executions)—looking for panics / crashes on weird bytes.
- API tests: happy path, bad JSON, wrong content type, oversized body, suggest off, suggest does not apply.
- Workbook tests: sheets exist, formulas present, `SafeText` on `=1+1`, pivot when there’s data.
- Vitest for the session helpers (grouping exceptions, movement messages).

**Manual / browser**

Headless Chrome against the real UI: upload fixture → see 46/18/28 → fix row 45 → recheck → download → open the zip and confirm Summary formulas and sheets. Screenshots and the write-up live in `docs/how-i-tested.md`.

**Demo script if they want a live walk**

1. Open the Fly URL.  
2. Upload `employees_messy.csv`.  
3. Point at first review row (missing id).  
4. Fix E044 / row 45 → recheck.  
5. Export; open Summary; show a formula and the disclaimer.

---

## A6. Closing the coding-project section (30 seconds)

**Say:**

> So the through-line is: honest parsing, explicit rules tied to IRS monthly measurement where they apply, no silent resolution of conflicts, a human review loop, and an Excel artifact whose numbers you can recalculate. The rest of ACA filing complexity is intentionally not faked.

Then pause. Let Blake steer.

---

# Part B — AI usage and verification (about 5–8 minutes, or woven into Part A)

## B1. Your stance in one breath

**Say:**

> An agent wrote most of the scaffolding—packages, first-pass tests, UI, excelize wiring. I did not treat that as finished work. I treated it like a fast junior: great for typing and exploring APIs, not allowed to set product policy or change data without a person in the loop.
>
> I trust the **rules I wrote down** and the **tests and fixtures that lock those rules**. I do not trust “the model said so.”

---

## B2. How you actually worked with the agent (process)

Walk this as a timeline:

1. **Research and decisions first**  
   IRS pages, CSV/RFC behavior, OWASP upload + injection, excelize docs. Decisions like “hold all duplicates” and “MDY only” were mine before generation.

2. **Agent implements under those constraints**  
   “Build parse/validate with these rules… workbook with formulas… Svelte review that rechecks through the API…”

3. **I read the diff against the decisions**  
   Especially: anything that drops rows, picks winners, mutates on suggest, soft-parses dates, or routes HTTP broadly.

4. **Automated gates**  
   vet, race tests, fuzz, golden file, Vitest.

5. **Black-box proof**  
   Real upload, real edit, real xlsx open—not just green tests.

6. **Write down where the agent was wrong**  
   README calls out three bugs on purpose. That’s part of the verification story, not a confession to hide.

---

## B3. Three agent mistakes — tell them as stories (this is gold)

### Bug 1 — CLI `-o` after the filename

**What I wanted:** `audit employees.csv -o out.xlsx`  
**What Go’s flag package does:** stops parsing flags at the first non-flag word. So after `employees.csv`, it ignored `-o`.  
**What the agent shipped:** “documented” that usage, but the binary didn’t honor it.  
**How I caught it:** Ran the documented command; output path was wrong / default.  
**Fix:** `normalizeArgs` pulls `-o` (and friends) before the filename, then hands the list to `flag`. Tests run both `file -o out` and `-o out file`.

**Plain English moral:** Agents copy common CLI patterns and docs; they don’t always reconcile them with library quirks. I verify the command I tell reviewers to run.

### Bug 2 — Pivot range quoting

**What I wanted:** A real Excel pivot over Clean Data.  
**What went wrong:** The sheet range string quoted the sheet name incorrectly, so the pivot was useless/broken.  
**How I caught it:** Opened the workbook / asserted pivot tables in tests.  
**Fix:** Correct range quoting; test that a pivot exists when there is clean data, and a clear note when there isn’t.

**Moral:** Spreadsheet APIs are stringly typed. Tests that only check “file writes without error” aren’t enough—inspect the artifact.

### Bug 3 — Catch-all `GET /` ate API routes

**What I wanted:** Serve the built UI from the same Go process.  
**What the agent did:** A broad file-server mount that could answer paths meant for `/api/...`.  
**How I caught it:** Hitting API paths didn’t behave like the API.  
**Fix:** Mount UI only on exact `/` and `/assets/`. API routes registered explicitly with Go 1.22 method patterns (`POST /api/validate`, etc.).

**Moral:** “Serve the SPA” is a classic footgun. Routing is part of product behavior; I test method-not-allowed and path separation.

---

## B4. Hard lines the agent is not allowed to cross

Memorize these three:

1. **Do not choose which duplicate to keep.**  
2. **Do not apply an LLM suggestion by itself.** Suggest returns JSON; the UI copies into a field only on click; recheck still required. Test name: `TestSuggestDoesNotApplyTheValue`.  
3. **Do not invent compliance answers** (look-back, penalties, 1095 codes) from incomplete columns.

Also: if the model returns prose instead of JSON for suggest, we reject it. Empty or huge suggestions rejected. Suggest with no API key → feature off (404), not a half-working button.

---

## B5. Verification checklist you can recite

When asked “how do you verify AI-generated code?”:

1. Diff vs written product rules.  
2. Table tests + golden fixture for the messy CSV.  
3. Fuzz parsers / anything that touches untrusted bytes.  
4. Race detector.  
5. HTTP edge cases (size, content type, methods).  
6. Open the binary artifact (xlsx) and check formulas / safe text.  
7. Drive the UI once the way a person would.  
8. Negative tests for dangerous features (suggest doesn’t mutate; duplicates not auto-resolved).

**Closer:**

> AI sped up building this. The correctness story is human judgment on scope, deterministic validation in Go, and a harness that fails when those rules drift.

---

# Part C — Timed talk track (pick one)

## Option 1 — Mostly project (15 min)

| Min | Topic |
| --- | --- |
| 0–1 | What it is + live demo pointer |
| 1–3 | Why messy hours matter; 130-hour rule; what I refused to fake |
| 3–7 | Life of a file: parse → validate → review → Excel |
| 7–12 | Judgment calls: duplicates, dates, hours, formulas, security, packages |
| 12–14 | How I tested (auto + browser + fixture numbers) |
| 14–15 | Bridge: “Most of it was agent-assisted; here’s how I caught mistakes…” (short) or stop for questions |

## Option 2 — Project + AI split (15–18 min)

Use Part A through A5 (~12 min), then Part B (~5 min) with the three bug stories.

## If they interrupt

- **“Show me”** → jump to A5 demo script.  
- **“Why Go?”** → straightforward errors, great CSV/HTTP stdlib, table tests + fuzz, excelize for xlsx; Svelte for a small reactive review UI without a heavy SPA stack.  
- **“Would you ship this?”** → Not as filing software. As an internal cleanup assist: add auth, persistence, look-back with real inputs, virus scanning, audit logging—keep the “no silent guesses” core.  
- **“Where did AI fail?”** → Bug stories B3; don’t be vague.

---

# Part D — Pocket examples (memorize 5)

1. **46 / 18 / 28** — fixture headline counts.  
2. **Row 45 / E044** — `nope` / `perhaps` → `40` / `yes` → moves to clean.  
3. **E037 duplicates** — both held.  
4. **E009 `=1+1`** — valid name; workbook stores with leading quote.  
5. **130 hours** — IRS monthly full-time; coverage `no` → gap flag, still clean.

---

# Out of scope for this doc

Excel interview skills test and TradeSiteUsa live elsewhere. This file is only what you can defend from **employee-hours-audit**.
