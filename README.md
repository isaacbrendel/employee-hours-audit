# Employee hours audit

A Go service reads a messy monthly-hours CSV, keeps every bad row in a review queue with a plain-English reason, and writes a workbook whose totals are Excel formulas. The Svelte page is the reviewer: upload, edit, recheck, download.

![Upload screen](docs/screenshots/01-upload.png)

It is a demo of data validation and an internal report. It is not tax, legal, or ACA filing advice, and it does not produce Form 1094-C or Form 1095-C.

This is Option C, a Go API and a Svelte front end. It also covers the CSV cleanup and the Excel summary, formulas, and chart from Options B and D.

## Run the CLI

```
go run ./cmd/audit testdata/employees_messy.csv -o /tmp/hours-audit.xlsx
```

A successful audit exits 0 even when some rows are in review. A missing file or a file that cannot be parsed exits 1. Wrong usage exits 2. `go run` itself prints `exit status 2` and returns 1 when the program exits 2. Build it if you need the process status:

```
go build -o audit ./cmd/audit
./audit testdata/employees_messy.csv -o /tmp/hours-audit.xlsx
```

## Run the server and the page

From the repository root:

```
go run ./cmd/server
```

That listens on `:8080` (override with `PORT`). If `web/dist` exists, the same process serves the built page at `/`.

In another terminal:

```
cd web
npm install
npm run dev
```

Vite serves the page and proxies `/api` and `/healthz` to `http://localhost:8080`. The default port is 5173. On this machine that port was already in use, so `npx vite --port 5174 --strictPort` was used instead.

```
cd web && npm test
cd web && npm run build
```

Suggestions are off unless `ANTHROPIC_API_KEY` is set in the server process. Copy `.env.example` to `.env` locally if you want them. The key is never written into the row; the reviewer accepts or rejects the value. Do not commit `.env`.

## Approach

`internal/record` parses with `encoding/csv`, trims, and validates. Anything that is not in an explicit date list, a plain decimal, or a known coverage word is an exception. Duplicates for the same employee and month are all held. A month at 130 or more hours is marked full-time (IRS monthly measurement method). A full-time month with coverage not offered is a flag on the clean row, not a rejected row and not a penalty determination.

`internal/report` uses excelize. Summary counts are formulas against Clean Data and Exceptions. Text cells that start with `=`, `+`, `-`, `@`, tab, CR, LF, or the full-width forms of those characters get a leading quote. Clean hours and coverage stay real numbers and booleans so the formulas can see them.

The HTTP API is `net/http` only and stores nothing. The page holds the working set and sends it back when you recheck or export.

Sources and the list of what was adopted, tightened, or left out are in [docs/research-notes.md](docs/research-notes.md).

## Left out on purpose

Look-back measurement, ALE status, Form 1095-C codes, affordability, and penalty amounts. The columns to support them are not in the file. A database, logins, and antivirus scanning. The upload cheat sheet's authentication guidance is unmet. `//go:embed` of the page: embed paths cannot use `..`, and `web/dist` is gitignored, so the server serves that directory from disk when it exists. An LLM that applies its own fix. Suggestions are a hint, and only when a key is present.

## Tests

`go vet ./...` was clean. `go test ./... -race -cover -v` passed. Statement coverage for `internal/record` was 93.6%. Module coverage was 79.1%. `FuzzParse` ran 60 seconds and passed (1,926,987 executions). Vitest ran 6 tests and passed. The planted CSV problems are listed in [testdata/README.md](testdata/README.md). The transcripts are in [docs/test-results.md](docs/test-results.md). The commands and the browser pass are in [docs/how-i-tested.md](docs/how-i-tested.md).

Manual check, in short: upload the fixture, confirm 46 submitted / 18 clean / 28 in review, correct row 45 to hours `40` and coverage `yes`, recheck, see that row move and row 31 stay, download the workbook, and confirm the COUNTIF and the quoted `=1+1` name.

## Where an agent helped, and where it had to be corrected

An agent wrote most of this repo, including the table tests and the workbook. These places needed a fix before they were true:

- The standard flag parser stops at the first filename, so `audit file.csv -o out.xlsx` ignored `-o` until the arguments were reordered. Both orders are tested.
- excelize rejected a pivot range that already quoted `'Clean Data'`. The range is `Clean Data!A1:E{n}`. Formulas still quote the sheet name themselves.
- A `GET /` pattern swallowed `GET /api/validate` and returned 404. The page is mounted at `GET /{$}` and `GET /assets/`, so the API returns 405 for the wrong method.
- `Review.svelte` imported `./lib/types` from inside `src/lib`. The production build failed until the imports were `./types` and `./api`.
- A leap-day example used a hire date after the reported month, so the test failed for the hire-date rule rather than the calendar rule.
- A server draft called `suggest.FromEnv` before that function existed. It was removed so the binary compiled, then added with the client and tested against a fake HTTP server, not the live API.
