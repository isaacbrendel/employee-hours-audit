# How I tested

29 September 2026. Go was `go version go1.26.5 darwin/amd64`. Port 5173 was already taken by another project on this machine (`ConstructionBrain` Vite), so the dev server was started on 5174. There is no browser automation in the editor, so the clicks below were driven with headless Chrome (`puppeteer-core`, installed in `/tmp`, not added to this repo) against the running app.

## Automated

Saved in full in [test-results.md](test-results.md).

```
go vet ./...
go test ./... -race -covermode=atomic -coverprofile=/tmp/hours.cover -count=1 -timeout 120s -v
go tool cover -func=/tmp/hours.cover
go test ./internal/record -fuzz=FuzzParse -fuzztime=60s
```

`go vet` exited 0 with no output. Every package in the race run passed. `internal/record` was 93.6% statement coverage. The module total was 79.1%. FuzzParse passed after 60 seconds (1,926,987 executions on the last minute mark). No panic.

```
cd web && npx vitest run
```

Vitest 4.1.11: 1 file, 6 tests, all passed.

`npm run build` in `web/` produced `dist/index.html` plus the CSS and JS assets (Vite 8.3.1).

## CLI

```
go build -o /tmp/hours-audit ./cmd/audit
/tmp/hours-audit testdata/employees_messy.csv -o /tmp/hours-audit.xlsx
```

Exit 0. Stderr: `warning: ignored extra column "Department"`. Stdout:

```
Rows submitted:   46
Clean rows:       18
Rows with errors: 28
Exception items:  34
Full-time months: 5
Coverage gaps:    1
Total hours:      1816
Wrote /tmp/hours-audit.xlsx
```

The same binary with no arguments exited 2 and printed the usage line. A missing path exited 1 and printed `read /tmp/does-not-exist.csv: open /tmp/does-not-exist.csv: no such file or directory`.

`go run ./cmd/audit` on the fixture printed the same summary and `Wrote` path, exit 0. `go run ./cmd/audit` with no file printed `exit status 2` on stderr and the `go` tool itself returned 1. The program's exit code is 2. `go run` wraps a non-zero program status. The tests call `run` directly so they see 2.

## API

Server from the repo root, so it can see `web/dist`:

```
go run ./cmd/server
```

Log: `listening on :8080`. `ANTHROPIC_API_KEY` was not set.

| Request | Result |
| --- | --- |
| `GET http://127.0.0.1:8080/healthz` | 200 `{"status":"ok","suggest":false}` |
| `GET http://localhost:5174/healthz` | 200, same body, through the Vite proxy |
| `POST /api/validate` multipart `file` = `testdata/employees_messy.csv` | 200, summary input 46, clean 18, exception items 34, rows with errors 28, full time 5, part time 13, coverage gaps 1, total hours 1816, warning `ignored extra column "Department"` |
| `POST /api/validate` JSON `{}` | 400 `{"error":"JSON body needs a rows array"}` |
| `POST /api/validate` `Content-Type: text/plain` | 415 `{"error":"Content-Type must be application/json or multipart/form-data"}` |
| multipart file named `not.csv.xlsx` | 415 `{"error":"upload a .csv file"}` |
| body larger than 1 MiB | 413 `{"error":"file is larger than 1 MiB"}` |
| `GET /api/validate` | 405, `Allow: POST` |
| `POST /api/suggest` | 404 `{"error":"suggestions are off because ANTHROPIC_API_KEY is not set"}` |
| `POST /api/report` JSON for E009, name `=1+1`, 10 hours, coverage yes | 200, `Content-Disposition: attachment; filename="hours-audit.xlsx"`, `X-Content-Type-Options: nosniff` |

The report zip contained sheets Summary, Clean Data, Exceptions, and Pivot, `fullCalcOnLoad="true"`, a pivot cache, and a COUNTIF formula.

`GET /` on the Go server returned the built `index.html`. `GET /assets/index-Cdjkem0e.js` was 200.

## Browser

Dev server:

```
cd web && npx vite --port 5174 --strictPort
```

Headless Chrome opened `http://localhost:5174/`.

1. The upload page showed the file field labeled "Hours CSV", the three steps, and Review and Export disabled until a file was checked. Screenshot: [screenshots/01-upload.png](screenshots/01-upload.png). A 390px-wide viewport stayed usable: [screenshots/01-upload-narrow.png](screenshots/01-upload-narrow.png).
2. The file input was given `testdata/employees_messy.csv` and Check this file was clicked. The page showed `46 rows submitted. 18 clean. 28 in review.` Counts: submitted 46, clean 18, in review 28, full time 5, coverage gaps 1, and the Department warning. The first review row was line 20, name "Missing Id", reason `employee_id is required`. Screenshot: [screenshots/02-review.png](screenshots/02-review.png).
3. Row 45 (E044, hours `nope`, coverage `perhaps`) was edited to hours `40` and coverage `yes`, then Recheck rows was clicked. Status became `Row 45 moved to clean data. 27 rows are still in review.` Counts became clean 19 and in review 27. The hours field for row 45 was gone. The hours field for row 31 (E030, hours `abc`) was still there. Screenshot: [screenshots/03-review-after-fix.png](screenshots/03-review-after-fix.png).
4. Continue to export showed clean 19, exception items 32, coverage gaps 1. Download workbook returned HTTP 200 and 17,412 bytes. Screenshot: [screenshots/04-export.png](screenshots/04-export.png).

I opened that zip. Workbook sheets: Summary, Clean Data, Exceptions, Pivot. `fullCalcOnLoad` is set. A chart part is present. Shared strings include `Two Problems` (E044) and `&#39;=1+1` (the apostrophe-prefixed name, not a formula). Summary contains `COUNTIF('Clean Data'!$G$2:$G$20,TRUE)`, which is the 19 clean rows.

The same Chrome profile opened `http://127.0.0.1:8080/`. The upload screen rendered (the file label was present). The pixels matched the Vite upload screenshot, so it is not stored twice.

Chrome logged one resource 404 during the first load. `GET http://localhost:5174/favicon.ico` is 404. `web/index.html` now has an empty icon so the browser does not have to request that path. I did not re-run the click-through after that one line.

I did not call the real Messages API. The suggestion client is tested against `httptest` only. The running server had no key, and Suggest a fix was not on the review page.
