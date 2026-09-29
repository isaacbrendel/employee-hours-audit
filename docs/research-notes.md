# Research notes

Demo only. This project is not tax, legal, or ACA compliance advice, and it does not prepare Form 1094-C or Form 1095-C.

Sources below are pages read for this project. Where a fact could not be verified, it says so.

## ChannelBound

Verified from the company's site and a 2026 press release:

- [ChannelBound](https://www.channelbound.com/) describes itself as a technology consulting firm and lists compliance work that includes ACA reporting through its ACAPrime service, ACA look-back measurement implementation, ACA audit and IRS response, ICHRA reporting, RxDC reporting, EEOC reporting, and PCORI fee filing. The same page lists data cleaning and data warehousing. A customer quote (Rodney Graham, Driving Ambition) describes payroll data spread across legacy systems and spreadsheets that ChannelBound cleaned and filed.
- [About](https://www.channelbound.com/about/) lists Indianapolis headquarters at 1311 W. 96th Street, Suite 170, and managing partner Brett Bussell (computer engineering, enterprise systems consulting, director of ACA compliance services).
- [Privacy](https://www.channelbound.com/privacy/) says ChannelBound, LLC operates channelbound.com, acaprime.com, and related domains.
- [Daybright Financial, 13 August 2026](https://www.daybright.com/news/daybright-financial-welcomes-acaprime-strengthening-aca-compliance-capabilities/) announced that ACAPrime joined Daybright. The release says ACAPrime was founded in 2014, is headquartered in Indianapolis, and handles managed ACA reporting, eligibility, filing, corrections, and IRS penalty response. Brett Bussell is quoted as founder and managing partner. The release says ACAPrime's management, staff, and client coverage stay in place, inside Daybright Broker Solutions.
- [ACAPrime's LinkedIn page](https://www.linkedin.com/company/acaprime) titles the company "ACAPrime (ChannelBound LLC)" and lists 1094-C/1095-C preparation, look-back measurement tracking, and IRS letter response. LinkedIn is a secondary source. The Daybright release and channelbound.com are the ones this project treats as primary.

Not verified: a public job posting that says "straightforward, maintainable solutions," names Go and Svelte, or describes the junior developer role. Those words are from the assignment brief, not from a page I could find. I did not invent a stack, a headcount, or a client list beyond the pages above.

What that work implies for this demo: hours in, a person checking the rows that do not parse, and a workbook a reviewer can defend. Look-back measurement is part of their real service. This demo does not implement it.

## ACA hours and reporting

Primary sources:

- [IRS, Identifying full-time employees](https://www.irs.gov/affordable-care-act/employers/identifying-full-time-employees). A full-time employee, for the employer shared responsibility provisions, is an employee employed on average at least 30 hours of service per week, or 130 hours of service per month. Under the monthly measurement method the employer looks at each month and asks whether the employee has at least 130 hours of service. The look-back measurement method determines status in a stability period from hours in an earlier measurement period. Hours of service include hours paid for work and hours paid for vacation, holiday, illness, layoff, jury duty, military duty, or leave, with exclusions the page lists (volunteers, federal work-study, certain religious-order work, non-U.S.-source compensation). The page points to regulation section 54.4980H-3.
- [IRS, Questions and answers on employer shared responsibility](https://www.irs.gov/affordable-care-act/employers/questions-and-answers-on-employer-shared-responsibility-provisions-under-the-affordable-care-act). Applicable large employers report whether they offered coverage on Form 1094-C and Form 1095-C. The same page states the 30-hour / 130-hour full-time definition used above.
- [IRS, Questions and answers on Form 1094-C and Form 1095-C](https://www.irs.gov/affordable-care-act/employers/questions-and-answers-about-information-reporting-by-employers-on-form-1094-c-and-form-1095-c). Employers subject to section 4980H use those forms to report offers of coverage and, for self-insured plans, enrollment. An ALE member generally files Form 1095-C for each employee who was full-time for any month.
- [2025 Instructions for Forms 1094-C and 1095-C](https://www.irs.gov/pub/irs-pdf/i109495c.pdf). The instructions say to count full-time employees using the section 4980H definition, not some other company definition, and they give a look-back example of an employee who averaged over 130 hours of service per month during a measurement period.

Used in the demo:

- A month with at least 130 hours is flagged full-time. That is the monthly measurement method only.
- A full-time month with coverage not offered is flagged as a coverage gap. The flag is a review aid. It is not a determination that the employer owes a payment under section 4980H. Offer of "minimum essential coverage," affordability, dependents, and the limited non-assessment period are not in the input columns, so the tool does not pretend to know them.

Not used, on purpose: look-back measurement, ALE status, full-time-equivalent math (the IRS Q&A describes adding non-full-time hours capped at 120 and dividing by 120), Form 1095-C line codes, and penalty amounts.

## CSV

- [RFC 4180](https://www.rfc-editor.org/rfc/rfc4180) is informational. Section 2 says records are lines separated by CRLF, a header is optional, each record should have the same number of fields, spaces are part of a field, and commas, quotes, and line breaks inside a field are quoted, with quotes escaped by doubling them. Section 3 registers `text/csv` and says that if the optional `header` parameter is absent, the implementation has to decide whether a header is present. Section 2 also says there is no single formal specification, which is why real files diverge.
- [encoding/csv](https://pkg.go.dev/encoding/csv) documents the Go reader as RFC 4180 with a few stated differences. The package comment in Go 1.26 says carriage returns before newlines are removed, blank lines are ignored, and a line that contains only whitespace is not a blank line. `Read` returns the record together with `ErrFieldCount` when the field count changes, so a short row can be reported and the parser can continue. `FieldPos` reports the 1-based line and column of a field in the record just read.

Real files also show up with a UTF-8 BOM, a header whose capitalization does not match, a trailing blank line, and a quoted comma. Those are planted in `testdata/employees_messy.csv` and covered by tests. I did not find a primary source that lists "inconsistent casing" as a standard. It is a property of files exported from payroll systems, which is why the header match is case-insensitive. That choice is ours, and it is documented.

## Uploads and formula injection

- [OWASP File Upload Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/File_Upload_Cheat_Sheet.html) says to allow only the extensions the feature needs, not to trust `Content-Type`, to limit file size, and not to store uploads inside the web root under a user-supplied name. It also lists authentication. This demo is stateless and has no accounts. Uploaded bytes are parsed and discarded. That removes the storage problem. It does not satisfy the cheat sheet's call for authentication, and the README says so.
- [OWASP CSV Injection](https://owasp.org/www-community/attacks/CSV_Injection) says a spreadsheet may treat a cell that starts with `=`, `+`, `-`, `@`, tab, carriage return, line feed, or the full-width forms `＝` `＋` `－` `＠` as a formula. One mitigation it describes is to prepend a single quote so the cell is text. The same page warns that quote-stripping can undo that mitigation for CSV files that Excel saves and reopens, and that there is no universal sanitization. This project writes `.xlsx` cells, not a CSV export. Untrusted text is still prefixed when it starts with one of those characters, and tests read the cell back.

## Excel

- [excelize](https://github.com/xuri/excelize/v2) is the library that writes xlsx. The module in this repo is v2.10.1. I pinned that release because its `Chart.Title` field is a `[]RichTextRun` and its `AddChart` takes `*Chart`. The [v2.11.0 notes](https://xuri.me/excelize/en/releases/v2.11.0.html) describe breaking changes to `AddChart`. I did not need those changes.
- [excelize charts](https://xuri.me/excelize/en/chart.html) and the v2.10.1 `AddChart` signature show a chart whose series point at sheet ranges.
- [excelize pivot tables](https://xuri.me/excelize/en/pivot.html) show one `AddPivotTable` call with a data range, row fields, and a sum field. That is the feature used here. It does not require a second library.
- Formulas are written with `SetCellFormula`. Summary counts use `COUNTIF`, `COUNTIFS`, `COUNTA`, and `SUM` against Clean Data. Per-row full-time and coverage-gap cells are formulas (`=E2>=130`, `=AND(G2=TRUE,F2=FALSE)`), so the workbook recomputes them. excelize does not evaluate formulas. Tests check that the stored formula is a formula and that its range matches the written rows. They do not claim Excel's calculation engine was executed.

## Go

- [Effective Go](https://go.dev/doc/effective_go) is the style baseline: straightforward control flow, errors returned and checked, packages small and focused.
- [Table-driven tests](https://go.dev/wiki/TableDrivenTests), [testing](https://pkg.go.dev/testing) subtests, and [fuzzing](https://go.dev/doc/security/fuzz/) are the test style. `go test -race` is part of the recorded run.
- [Go 1.22 routing](https://go.dev/blog/routing-enhancements) is why the server uses `http.ServeMux` patterns such as `POST /api/validate` and no router library.
- [embed](https://pkg.go.dev/embed) was considered for a single binary. Patterns cannot contain `..`, so the server cannot embed `web/dist` without copying the build into the server package. Build output is gitignored. The dev server proxies `/api` instead. See decisions.

## Svelte and Vite

- [Svelte documentation](https://svelte.dev/docs) and [Vite](https://vite.dev/guide/) are the frontend baseline. The app is Vite + Svelte + TypeScript, with no component library. The dev server proxies API calls so the browser does not need CORS.

## Decisions

### Adopting

- Monthly full-time flag at 130 hours, cited in code, because that is the IRS monthly measurement method. Coverage gaps are flags on clean rows, not rejected rows. A valid "no" is data. Dropping it would hide the case the reviewer most needs to see.
- `encoding/csv` for real parsing. A hand-rolled splitter would mishandle quotes. Short records become row exceptions because `Read` returns the record and `ErrFieldCount`. A broken quote stops the file. After an unclosed quote the rest of the file is not aligned, and guessing the rest would invent rows.
- Row numbers from `FieldPos`, which is the physical line. Blank lines are not records. The fixture's trailing blank lines produce no exception.
- An explicit date list: `YYYY-MM-DD`, `YYYY/MM/DD`, `MM/DD/YYYY`, `M/D/YYYY`, `DD-Mon-YYYY`. Slash dates are month/day/year. This is a US reporting demo, and `13/01/2024` fails instead of being reread as 13 January. Anything that does not round-trip through the layout that parsed it is an exception, so `02/31/2024` is not normalized to a later day.
- Hours must match a plain decimal (`160`, `37.5`). Commas and scientific notation are exceptions. Negative hours are an exception. Hours above 744 are an exception. 744 is 31 times 24, a physical bound for this demo, not an IRS rule. Zero hours is allowed. It is not the same as a blank cell.
- Coverage accepts only `true/false`, `t/f`, `yes/no`, `y/n`, and `1/0`, case-insensitive, after trimming. Every other token is an exception, including `maybe` and `offered`.
- Duplicate `employee_id` + `month`: every copy is held. Keeping the first would be a guess. IDs are case-sensitive. `E037` and `e037` are different. That is a choice. The tool does not know the employer's identifier rules.
- Missing required fields are exceptions. One row can carry several, so a reviewer can fix them in one pass.
- Header row required. RFC 4180 leaves that decision to the reader when the MIME header parameter is absent. Payroll files that shift columns cannot be mapped safely without names. Header match is trimmed, lowercased, and spaces become underscores, so `Employee ID` and `hours_worked` both map. Values are not case-folded except coverage.
- Surrounding whitespace is trimmed before validation. RFC 4180 says spaces are part of a field. Payroll exports pad them. Trimming is an explicit normalization, not a silent drop of the row. Internal spaces in a name stay.
- Extra columns produce a warning and are not copied into the clean row. They are not deleted from the source file. The warning names the column.
- UTF-8 required, BOM stripped. A file that is not UTF-8 errors. That is stricter than RFC 4180, which allows other charsets, and it avoids mojibake.
- Formula prefixes neutralized on every text cell written to the workbook, including exception raw values such as `-4` and `=1+1`.
- Summary measures are formulas. The count of rows submitted is a typed value, labeled as one, because a source row with two problems occupies two exception lines and the file's trailing blanks are not rows.
- `net/http` only, Go 1.22 patterns, 1 MiB limit, `.csv` allowlist, content type checked and then the bytes parsed anyway.
- Stateless API. The browser holds the working set. Nothing is stored, which matches the part of the upload cheat sheet this demo can actually meet.
- Table tests, a golden JSON file, `go test -race`, and a 60-second fuzz of the parser.

### Improving on the cited practice, and why

- OWASP's CSV guidance is about CSV text. The deliverable is xlsx. Neutralizing inside the xlsx cell still matters because a string that starts with `=` can be what a reviewer expects to see as text. Prefixing a quote is the mitigation OWASP lists, applied at the cell, with a test that reads the workbook back. I am not claiming it is safe for every spreadsheet program. OWASP says no mitigation is.
- excelize pivot tables are one call, so the workbook includes a real pivot of hours by month. The summary the reviewer argues from is still the formula block, not the pivot. If a future excelize release breaks the pivot, the formula block is the one the tests lock.

### Skipping, and why

- Look-back measurement, 1095-C codes, affordability, and penalty math. The columns are not there, and a wrong answer would look more official than a blank one.
- A database, accounts, and antivirus scanning. The assignment asks for a review queue and an export, and it says to keep the tool stateless. OWASP's authentication guidance is unmet and stated as unmet.
- Embedding the UI in the Go binary. `//go:embed` paths cannot climb out of the package with `..`, and build output is gitignored. Vite proxies `/api` in development. The server serves `web/dist` when that directory exists.
- A second UI framework and a CSV writer of our own.
- An LLM that files the correction by itself. `POST /api/suggest` calls the [Messages API](https://docs.anthropic.com/en/api/messages) only when `ANTHROPIC_API_KEY` is set: `POST /v1/messages`, header `anthropic-version: 2023-06-01`, header `x-api-key`. The default model id is `claude-sonnet-5`, which appears in that page's model enum. The page's own curl example uses `claude-opus-5`. `ANTHROPIC_MODEL` overrides the default. The reply has to be JSON with one replacement value. Prose is an error. The row is not updated until a person uses the value and rechecks.
