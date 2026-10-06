# How I built the employee hours audit (study notes)

Read this like a blog post. The goal is that you understand the project well enough to explain it in **your** words—not to memorize a script.

Live demo: https://employee-hours-audit.fly.dev/  
Sample file: `testdata/employees_messy.csv` (expect about **46** rows in, **18** clean, **28** needing review)

---

## What this project is

I built a small tool that takes a messy monthly employee-hours CSV and helps a person clean it.

It does three things:

1. **Reads** the file and decides which rows look trustworthy and which ones a person has to look at.
2. **Lets a person fix** the bad rows in a web UI, then runs the same checks again.
3. **Exports an Excel workbook** with clean data, a list of problems, and summary counts. Those summary counts are Excel *formulas* that read the data sheets—not numbers the server pasted in and hoped nobody would question.

The stack is Go on the backend (CLI + HTTP API) and Svelte on the front end. That was Option C on the assignment, and it also covers the CSV cleanup and Excel reporting pieces.

Important honesty up front: this is a **demo**. It is not tax advice, not legal advice, and it does not produce Form 1094-C or Form 1095-C. ACAPrime’s real work is much bigger. This project is a slice of the “messy payroll data in → something a reviewer can defend” problem.

---

## Why the problem looks like this

Companies that do ACA reporting often get hours data out of payroll systems as CSVs and spreadsheets. Those files are rarely clean. You get:

- blank employee IDs
- dates written five different ways
- hours like `abc`, `1,200`, or `1e2`
- coverage written as `maybe` instead of yes/no
- two rows for the same person in the same month
- names that start with `=` and look like Excel formulas
- extra columns nobody asked for

If software quietly “fixes” those things—keeps the first duplicate, caps weird hours, guesses what a date meant—it can hide the exact rows a human needed to see. For compliance-ish data, a wrong automatic answer is worse than an honest “this needs a person.”

So the product idea is simple:

> Parse carefully. Hold anything unclear. Let a human decide. Export something auditable.

---

## The one ACA rule I actually used

Under the IRS **monthly measurement method**, an employee with at least **130 hours of service in a calendar month** is full-time for that month. That is roughly 30 hours a week.

My tool flags full-time that way. If someone is full-time and coverage is `no`, I also flag a **coverage gap**. That flag is a review aid. It is **not** me saying the employer owes a penalty. I do not have enough columns (affordability, dependents, limited non-assessment periods, and so on) to make that call.

I deliberately did **not** build:

- look-back measurement (measure hours in an earlier window, lock status later)
- Form 1095-C line codes
- affordability tests
- penalty math

If someone asks why, the honest answer is: inventing official-looking answers from incomplete inputs would be more dangerous than leaving them out and saying so.

---

## What happens to a file, start to finish

### 1. Reading the CSV

Same core logic whether you use the CLI or the website.

- Read the file (with a size/row cap so a huge upload cannot take the process down).
- Strip a UTF-8 BOM if Excel stuck one on the front.
- Reject the whole file if it is not valid UTF-8. Mojibake that “almost works” is worse than a clear error.
- Parse with Go’s `encoding/csv`, not a homemade split-on-commas. Real files have quoted commas in names, Windows line endings, and blank lines.

**Headers:** I normalize names (trim, lowercase, spaces → underscores) so `Employee_ID`, `NAME`, and `Hours Worked` all map. Required columns must exist or the file fails. An extra column like `Department` becomes a warning and is ignored for the audit—not copied into clean data, and not a reason to reject the whole file.

**Row numbers:** Physical line numbers. Header is line 1. Blank lines are not records, so trailing blank lines do not create fake errors.

**Broken structure:**

- Wrong number of columns → that row is held for review. I do **not** slide fields sideways into the next person’s data.
- A broken quote (unclosed `"`) → stop the file. After that, the bytes are no longer aligned; continuing would invent garbage rows.

### 2. Validating each row

Every field has a rule. Failures become **exceptions**: which row, which field, why, and the raw values. One row can have several exceptions so a person can fix everything in one pass.

Clean rows get normalized (hire date to `YYYY-MM-DD`, hours to a number, coverage to true/false) and two derived flags: full-time and coverage gap.

Concrete examples from the sample file that are good to remember:

| Messy input | What happens |
| --- | --- |
| Missing employee ID | Held for review |
| Hire date `March 5th 2024` | Held—not in the accepted date list |
| `13/01/2024` | Held—slash dates are month/day/year; I do not reinterpret as 13 January |
| Hours `abc`, `-4`, `1,200`, `800` | Held—I do not guess or silently cap |
| Coverage `maybe` | Held—only clear yes/no-style values |
| Two rows for the same employee + month | **Both** held—I do not pick a winner |
| Name `=1+1` | Allowed as data; Excel export must store it as text, not a live formula |

### 3. Review in the browser

The UI has three steps: Upload → Review → Export.

You edit bad rows in the browser. When you hit **Recheck**, the browser sends the whole working set back to Go and runs the **same** validation. The front end does not reimplement the rules. That matters: one definition of “valid,” everywhere.

If an Anthropic API key is configured, you can ask for a suggested fix. Using the suggestion only fills the field. The row stays in review until you recheck. No key → suggestions are off.

### 4. Excel export

The workbook has Summary, Clean Data, Exceptions, and a Pivot sheet.

- Clean Data and Exceptions hold the rows.
- Most Summary metrics are formulas (`COUNTIF`, `SUM`, etc.) pointed at those sheets.
- “Rows submitted” is the one typed number, because source rows and exception *lines* are not one-to-one (one bad row can produce multiple exception items).
- Full-time and coverage-gap columns on Clean Data are formulas too, so Excel recalculates them.
- Text that looks like a formula (`=`, `+`, `-`, `@`, …) gets a leading quote so spreadsheet apps treat it as text.
- There is a clear disclaimer on Summary: demo only, not a filing.

The CLI writes the same kind of workbook: `audit file.csv -o report.xlsx`. Exit code 0 means “I wrote a workbook,” even if some rows still need review. Exit 1 is a read/parse failure. Exit 2 is bad usage.

---

## Design decisions (the ones worth defending in an interview)

These are the topics a CTO is likely to push on. For each one: what I chose, why, what I traded away, and what you might get asked.

### 1. Ambiguous data fails closed — a person decides

**Choice:** When the tool is unsure (duplicates, weird hours, ambiguous dates, unclear coverage), it holds the row. It does not auto-repair.

**Why:** This is compliance-adjacent data. Silent fixes create silent liability. Showing the conflict preserves evidence.

**Tradeoff:** More review work for the human. That is intentional for a cleanup tool.

**They might ask:** “Wouldn’t it be better to keep the latest duplicate automatically?”  
**Answer shape:** Maybe as a *suggested* default in a future product—with an audit log—but not as an invisible default. Choosing which record is true is a business judgment. The tool’s job is to surface the conflict.

### 2. One validation path for CLI, API, and recheck

**Choice:** `internal/record` owns the rules. The CLI, `POST /api/validate`, and the browser’s “Recheck” all go through that package. The UI does not have its own copy of “what counts as full-time.”

**Why:** Split-brain validation is how bugs ship—“looked fine in the UI, failed in the export,” or the reverse.

**Tradeoff:** Every change to a rule requires regenerating/fixtures and thinking about both CLI and web. Worth it.

**They might ask:** “Why not validate in the browser for snappiness?”  
**Answer shape:** You can add client-side hints later for UX, but the server (or shared library) remains the source of truth for anything that hits the workbook.

### 3. Stateless API — the browser holds the working set

**Choice:** The server stores nothing. Upload bytes are parsed and discarded. Edited rows live in the browser until export.

**Why:** Matches the assignment, reduces breach surface for a demo (no forgotten database of employee names), and keeps the architecture small.

**Tradeoff:** Refresh loses work. No multi-user review queue. No “come back tomorrow.”

**They might ask:** “How would you productionize this?”  
**Answer shape:** Add auth, encrypt data in transit and at rest, persist review sessions with access control and retention policy, audit who changed what, and still keep validation deterministic. Stateless was a scope choice, not a belief that HR data should float in localStorage forever.

### 4. Summary counts are Excel formulas, not pasted numbers

**Choice:** The workbook recomputes clean-row counts, full-time months, gaps, hours, and exception counts with formulas against the data sheets.

**Why:** A reviewer can change or filter data and still reason about the Summary tab. It is an auditability feature, not a flourish. Pasted numbers go stale the moment someone edits a cell.

**Tradeoff:** excelize does not run Excel’s calculation engine. My tests assert the *formula text and ranges* are correct; they do not claim Excel already calculated them in CI. I say that out loud.

**They might ask:** “Why not compute in Go and write values?”  
**Answer shape:** Go already computed them for the UI and CLI summary. The workbook audience is different: people who live in Excel and need the sheet to defend itself.

### 5. Domain scope is narrow on purpose

**Choice:** Implement monthly 130-hour full-time + coverage-gap flags. Skip look-back, 1095-C codes, affordability, penalties.

**Why:** Incomplete inputs + authoritative-looking outputs = dangerous. Scope discipline is a feature when the domain is regulated.

**Tradeoff:** The demo does not showcase the full ACAPrime surface area.

**They might ask:** “Do you understand look-back?”  
**Answer shape:** Yes, at a high level—hours in a measurement period determine status in a later stability period. I did not implement it because the CSV does not carry measurement/stability windows, and a fake implementation would look more “done” than it is. I would want product + compliance input before encoding that.

### 6. Security boundaries that match the threat of a public demo

**Choice:**

- 1 MiB upload limit (reject oversize; do not truncate mid-file)
- Prefer `.csv` uploads
- Check content type, but still parse bytes (do not trust headers alone)
- Neutralize spreadsheet formula injection on export
- No accounts in this demo (and the README admits that gap)

**Why:** Employee names and IDs are sensitive even in synthetic fixtures. Upload and export are the two places untrusted strings enter Excel-land.

**Tradeoff:** Without auth, anyone who can reach the demo can use it. Fine for a synthetic public demo; not fine for real employer files.

**They might ask:** “What is CSV injection?”  
**Answer shape:** If a cell starts with `=`, Excel may treat it as a formula. A name like `=1+1` or worse can become active content. I prefix risky text when writing xlsx cells. OWASP notes no mitigation is perfect for every spreadsheet tool; I still do the standard hardening and test it.

### 7. Package boundaries: rules, reports, HTTP, and AI stay separate

**Choice:**

| Package | Responsibility |
| --- | --- |
| `record` | Parse + validate only |
| `report` | Build the workbook |
| `api` | HTTP surface |
| `suggest` | Optional LLM suggestions; never mutates rows |
| `cmd/audit`, `cmd/server` | Wiring |
| `web/` | UI and session helpers |

**Why:** When rules live in one place, you can change HTTP or Excel without rewriting what “duplicate” means. When AI lives in `suggest`, it cannot casually become part of validation.

**They might ask:** “How do you keep an LLM from becoming the system of record?”  
**Answer shape:** Architecturally: suggest is optional, returns a proposal, and is tested to not apply values. Product-wise: a human must accept and recheck through deterministic rules.

### 8. AI is assistive, never authoritative

**Choice:** Suggestions are off unless a key is set. Even when on, the API returns a proposed value; the server does not write it into the row. The UI only fills a field if the person clicks to use it, and they still must recheck.

**Why:** LLMs are good at “this cell looks like it should be 160.” They are bad at being the compliance brain. Auto-apply would blur the line between assistance and decision-making.

**They might ask:** “How did you use AI while building this?”  
**Answer shape:** An agent wrote a lot of scaffolding. I set the product rules, reviewed diffs against those rules, and verified with tests plus a real browser pass. I also caught concrete agent mistakes (CLI flag order, pivot range quoting, a catch-all static route that could swallow API paths). Those are fixed and tested. I would not let an agent choose which duplicate to keep.

---

## How I know it works

**Automated:** table tests for parsing and rules; a golden JSON file for the messy sample CSV; race detector; fuzzing the parser for about a minute; API tests for bad uploads and the “suggest does not apply” behavior; workbook tests for formulas and safe text; a few front-end unit tests.

**Manual:** upload the sample file, fix row 45 (hours `nope` / coverage `perhaps` → `40` / `yes`), recheck, download the workbook, and confirm the sheets and formulas. That walkthrough is written up under `docs/how-i-tested.md`.

If you demo live: use the Fly URL, upload the sample CSV, show a review reason, fix row 45, export, open Summary and point at a formula and the disclaimer.

---

## Using AI on this project (study section)

### How I worked with it

1. I researched and decided the rules (130 hours, hold duplicates, date formats, no silent caps).
2. The agent implemented structure, tests, UI, Excel wiring under those constraints.
3. I reviewed the diff like I would a junior PR—especially anything that drops data, guesses, or widens HTTP routes.
4. Green tests were necessary but not sufficient; I still ran the real file through the UI and opened the xlsx.
5. I documented where the agent was wrong instead of pretending the first draft was clean.

### Three mistakes worth being able to explain

1. **`-o` after the filename** — Go’s flag parser stops at the first non-flag argument, so `audit file.csv -o out.xlsx` did not behave as documented until I normalized argument order and tested both orders.
2. **Pivot range quoting** — the workbook wrote, but the pivot range string was wrong; “file created” ≠ “artifact is correct.”
3. **Catch-all UI route** — serving the SPA too broadly could intercept API paths; UI is mounted only on `/` and `/assets/`, with API routes registered explicitly.

### Verification mindset

I do not “trust AI code.” I trust:

- written product rules
- tests and fixtures that encode those rules
- opening the real output
- keeping humans in the loop for judgment calls

---

## If they ask “what would you do next?”

Good answers sound concrete:

- Authentication and authorization before any real employer data
- Persisted review sessions with audit trail (who changed which field)
- Virus scanning / stronger upload controls
- Look-back measurement **after** the input model supports it
- Clearer separation of environments (demo synthetic data vs. customer data)
- Keep the core invariant: **deterministic validation; no silent resolution of conflicts**

---

## A few facts worth remembering

- Sample file: **46** submitted, **18** clean, **28** in review  
- Fix demo: **row 45** / E044  
- Duplicates: **E037** — both held  
- Formula-looking name: **E009** `=1+1`  
- Full-time threshold: **130** hours / month  
- Demo: https://employee-hours-audit.fly.dev/

---

## How to study this

1. Read the whole post once without trying to memorize.
2. Close it and explain out loud: what the tool does, what happens to a file, and three design decisions you care about.
3. Skim the design-decision section again and pick the tradeoffs you feel strongest about (fail closed, one validation path, formulas, AI not authoritative are the strongest set).
4. Click through the live demo once with the sample file so the story is tied to something you have seen.
