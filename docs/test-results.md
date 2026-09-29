# Test results

Recorded 2026-09-29 on this machine. `go version` printed `go version go1.26.5 darwin/amd64`.

Commands, in order:

```
go vet ./...
go test ./... -race -covermode=atomic -coverprofile=/tmp/hours.cover -count=1 -timeout 120s -v
go tool cover -func=/tmp/hours.cover
go test ./internal/record -fuzz=FuzzParse -fuzztime=60s -count=1
```

`go vet ./...` exited 0 and printed nothing.

`internal/record` statement coverage on that race run: **93.6%** (`ok github.com/isaacbrendel/employee-hours-audit/internal/record coverage: 93.6% of statements`). The whole module, from `go tool cover -func`, was **79.1%**. The lower number includes error paths in the workbook writer that the tests do not force.

The fuzz run finished `PASS` after 60 seconds. The last progress line was `elapsed: 1m0s, execs: 1926987`. It did not report a panic or a failing input.

## go test -race -cover -v

```
=== RUN   TestCLI
--- PASS: TestCLI (0.35s)
=== RUN   TestCLIUsage
--- PASS: TestCLIUsage (0.00s)
=== RUN   TestCLIMissingFile
--- PASS: TestCLIMissingFile (0.00s)
=== RUN   TestCLIBadHeader
--- PASS: TestCLIBadHeader (0.00s)
PASS
coverage: 84.3% of statements
ok  	github.com/isaacbrendel/employee-hours-audit/cmd/audit	2.774s	coverage: 84.3% of statements
	github.com/isaacbrendel/employee-hours-audit/cmd/server		coverage: 0.0% of statements
=== RUN   TestHealthAndFallback
--- PASS: TestHealthAndFallback (0.00s)
=== RUN   TestStaticUI
--- PASS: TestStaticUI (0.02s)
=== RUN   TestValidateJSON
--- PASS: TestValidateJSON (0.00s)
=== RUN   TestValidateRejectsBadJSONAndMissingRows
--- PASS: TestValidateRejectsBadJSONAndMissingRows (0.00s)
=== RUN   TestValidateCSVAndRejections
--- PASS: TestValidateCSVAndRejections (0.00s)
=== RUN   TestOversizedUpload
--- PASS: TestOversizedUpload (0.02s)
=== RUN   TestReportWorkbook
--- PASS: TestReportWorkbook (0.11s)
=== RUN   TestSuggestIsHiddenUntilConfigured
--- PASS: TestSuggestIsHiddenUntilConfigured (0.00s)
=== RUN   TestSuggestDoesNotApplyTheValue
--- PASS: TestSuggestDoesNotApplyTheValue (0.00s)
=== RUN   TestMethodNotAllowed
--- PASS: TestMethodNotAllowed (0.00s)
PASS
coverage: 81.7% of statements
ok  	github.com/isaacbrendel/employee-hours-audit/internal/api	4.368s	coverage: 81.7% of statements
=== RUN   TestMessyFixture
--- PASS: TestMessyFixture (0.00s)
=== RUN   TestGolden
--- PASS: TestGolden (0.01s)
=== RUN   TestParseShapes
=== RUN   TestParseShapes/empty
=== RUN   TestParseShapes/not_utf-8
=== RUN   TestParseShapes/missing_column
=== RUN   TestParseShapes/duplicate_header
=== RUN   TestParseShapes/empty_header_name
=== RUN   TestParseShapes/broken_quote_stops_the_file
=== RUN   TestParseShapes/header_only
=== RUN   TestParseShapes/bom_and_mixed_header_case
=== RUN   TestParseShapes/crlf_quoted_comma_and_trailing_blank_lines
=== RUN   TestParseShapes/short_row_is_not_mapped_onto_the_next_record
=== RUN   TestParseShapes/blank_line_in_the_middle_does_not_consume_a_data_row
=== RUN   TestParseShapes/extra_column_is_a_warning
--- PASS: TestParseShapes (0.00s)
    --- PASS: TestParseShapes/empty (0.00s)
    --- PASS: TestParseShapes/not_utf-8 (0.00s)
    --- PASS: TestParseShapes/missing_column (0.00s)
    --- PASS: TestParseShapes/duplicate_header (0.00s)
    --- PASS: TestParseShapes/empty_header_name (0.00s)
    --- PASS: TestParseShapes/broken_quote_stops_the_file (0.00s)
    --- PASS: TestParseShapes/header_only (0.00s)
    --- PASS: TestParseShapes/bom_and_mixed_header_case (0.00s)
    --- PASS: TestParseShapes/crlf_quoted_comma_and_trailing_blank_lines (0.00s)
    --- PASS: TestParseShapes/short_row_is_not_mapped_onto_the_next_record (0.00s)
    --- PASS: TestParseShapes/blank_line_in_the_middle_does_not_consume_a_data_row (0.00s)
    --- PASS: TestParseShapes/extra_column_is_a_warning (0.00s)
=== RUN   TestParseBytesBOM
--- PASS: TestParseBytesBOM (0.00s)
=== RUN   TestValidationRules
=== RUN   TestValidationRules/full_time_with_coverage
=== RUN   TestValidationRules/full_time_without_coverage_is_a_flag,_not_an_exception
=== RUN   TestValidationRules/just_under_130_is_part_time
=== RUN   TestValidationRules/exactly_130_is_full_time
=== RUN   TestValidationRules/zero_hours_is_valid
=== RUN   TestValidationRules/744_hours_is_the_last_accepted_value
=== RUN   TestValidationRules/above_744_is_not_capped
=== RUN   TestValidationRules/negative_hours
=== RUN   TestValidationRules/negative_zero
=== RUN   TestValidationRules/thousands_separator
=== RUN   TestValidationRules/scientific_notation
=== RUN   TestValidationRules/hours_text
=== RUN   TestValidationRules/missing_employee_id
=== RUN   TestValidationRules/missing_name
=== RUN   TestValidationRules/missing_hire_date
=== RUN   TestValidationRules/missing_month
=== RUN   TestValidationRules/missing_hours
=== RUN   TestValidationRules/missing_coverage
=== RUN   TestValidationRules/prose_date
=== RUN   TestValidationRules/day_greater_than_12_is_not_reread_as_DMY
=== RUN   TestValidationRules/impossible_calendar_day
=== RUN   TestValidationRules/slash_month_is_not_YYYY-MM
=== RUN   TestValidationRules/month_13
=== RUN   TestValidationRules/compact_month
=== RUN   TestValidationRules/coverage_maybe
=== RUN   TestValidationRules/coverage_offered_is_not_a_boolean
=== RUN   TestValidationRules/hired_after_the_reported_month
=== RUN   TestValidationRules/hired_during_the_reported_month
=== RUN   TestValidationRules/control_character_in_the_name
=== RUN   TestValidationRules/US_slash_date_normalizes
=== RUN   TestValidationRules/unpadded_slash_date_normalizes
=== RUN   TestValidationRules/year_slash_date_normalizes
=== RUN   TestValidationRules/day_month_name_date_normalizes
=== RUN   TestValidationRules/leap_day
=== RUN   TestValidationRules/leading_zeros_survive
--- PASS: TestValidationRules (0.01s)
    --- PASS: TestValidationRules/full_time_with_coverage (0.00s)
    --- PASS: TestValidationRules/full_time_without_coverage_is_a_flag,_not_an_exception (0.00s)
    --- PASS: TestValidationRules/just_under_130_is_part_time (0.00s)
    --- PASS: TestValidationRules/exactly_130_is_full_time (0.00s)
    --- PASS: TestValidationRules/zero_hours_is_valid (0.00s)
    --- PASS: TestValidationRules/744_hours_is_the_last_accepted_value (0.00s)
    --- PASS: TestValidationRules/above_744_is_not_capped (0.00s)
    --- PASS: TestValidationRules/negative_hours (0.00s)
    --- PASS: TestValidationRules/negative_zero (0.00s)
    --- PASS: TestValidationRules/thousands_separator (0.00s)
    --- PASS: TestValidationRules/scientific_notation (0.00s)
    --- PASS: TestValidationRules/hours_text (0.00s)
    --- PASS: TestValidationRules/missing_employee_id (0.00s)
    --- PASS: TestValidationRules/missing_name (0.00s)
    --- PASS: TestValidationRules/missing_hire_date (0.00s)
    --- PASS: TestValidationRules/missing_month (0.00s)
    --- PASS: TestValidationRules/missing_hours (0.00s)
    --- PASS: TestValidationRules/missing_coverage (0.00s)
    --- PASS: TestValidationRules/prose_date (0.00s)
    --- PASS: TestValidationRules/day_greater_than_12_is_not_reread_as_DMY (0.00s)
    --- PASS: TestValidationRules/impossible_calendar_day (0.00s)
    --- PASS: TestValidationRules/slash_month_is_not_YYYY-MM (0.00s)
    --- PASS: TestValidationRules/month_13 (0.00s)
    --- PASS: TestValidationRules/compact_month (0.00s)
    --- PASS: TestValidationRules/coverage_maybe (0.00s)
    --- PASS: TestValidationRules/coverage_offered_is_not_a_boolean (0.00s)
    --- PASS: TestValidationRules/hired_after_the_reported_month (0.00s)
    --- PASS: TestValidationRules/hired_during_the_reported_month (0.00s)
    --- PASS: TestValidationRules/control_character_in_the_name (0.00s)
    --- PASS: TestValidationRules/US_slash_date_normalizes (0.00s)
    --- PASS: TestValidationRules/unpadded_slash_date_normalizes (0.00s)
    --- PASS: TestValidationRules/year_slash_date_normalizes (0.00s)
    --- PASS: TestValidationRules/day_month_name_date_normalizes (0.00s)
    --- PASS: TestValidationRules/leap_day (0.00s)
    --- PASS: TestValidationRules/leading_zeros_survive (0.00s)
=== RUN   TestCoverageSynonyms
--- PASS: TestCoverageSynonyms (0.00s)
=== RUN   TestSeveralProblemsOnOneRow
--- PASS: TestSeveralProblemsOnOneRow (0.00s)
=== RUN   TestDuplicatesAreAllHeld
--- PASS: TestDuplicatesAreAllHeld (0.00s)
=== RUN   TestIdenticalDuplicatesAreStillHeld
--- PASS: TestIdenticalDuplicatesAreStillHeld (0.00s)
=== RUN   TestWhitespaceIsTrimmedNotRejected
--- PASS: TestWhitespaceIsTrimmedNotRejected (0.00s)
=== RUN   TestJSONRowNumberFallback
--- PASS: TestJSONRowNumberFallback (0.00s)
=== RUN   FuzzParse
=== RUN   FuzzParse/seed#0
=== RUN   FuzzParse/seed#1
=== RUN   FuzzParse/seed#2
=== RUN   FuzzParse/seed#3
=== RUN   FuzzParse/seed#4
--- PASS: FuzzParse (0.00s)
    --- PASS: FuzzParse/seed#0 (0.00s)
    --- PASS: FuzzParse/seed#1 (0.00s)
    --- PASS: FuzzParse/seed#2 (0.00s)
    --- PASS: FuzzParse/seed#3 (0.00s)
    --- PASS: FuzzParse/seed#4 (0.00s)
PASS
coverage: 93.6% of statements
ok  	github.com/isaacbrendel/employee-hours-audit/internal/record	2.177s	coverage: 93.6% of statements
=== RUN   TestSafeText
--- PASS: TestSafeText (0.00s)
=== RUN   TestWorkbook
--- PASS: TestWorkbook (0.18s)
=== RUN   TestEmptyWorkbook
--- PASS: TestEmptyWorkbook (0.01s)
PASS
coverage: 70.9% of statements
ok  	github.com/isaacbrendel/employee-hours-audit/internal/report	3.661s	coverage: 70.9% of statements
=== RUN   TestAnthropicSuggestParsesTextAndDoesNotChangeTheRow
--- PASS: TestAnthropicSuggestParsesTextAndDoesNotChangeTheRow (0.01s)
=== RUN   TestAnthropicRejectsProse
--- PASS: TestAnthropicRejectsProse (0.00s)
=== RUN   TestFromEnvWithoutKey
--- PASS: TestFromEnvWithoutKey (0.00s)
PASS
coverage: 69.7% of statements
ok  	github.com/isaacbrendel/employee-hours-audit/internal/suggest	2.810s	coverage: 69.7% of statements
```

## go tool cover -func

```
github.com/isaacbrendel/employee-hours-audit/cmd/audit/main.go:15:		main		0.0%
github.com/isaacbrendel/employee-hours-audit/cmd/audit/main.go:19:		run		86.8%
github.com/isaacbrendel/employee-hours-audit/cmd/audit/main.go:71:		normalizeArgs	83.3%
github.com/isaacbrendel/employee-hours-audit/cmd/server/main.go:19:		main		0.0%
github.com/isaacbrendel/employee-hours-audit/cmd/server/main.go:43:		listenAddr	0.0%
github.com/isaacbrendel/employee-hours-audit/cmd/server/main.go:54:		uiDir		0.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:37:		Handler		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:47:		health		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:54:		validate	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:62:		report		83.3%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:79:		suggest		57.1%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:111:		auditRequest	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:129:		auditJSON	81.8%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:148:		auditCSV	70.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:176:		mountUI		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:191:		readLimited	90.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:209:		writeReadError	75.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:217:		mediaType	75.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:225:		csvType		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:234:		writeError	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/api/api.go:238:		writeJSON	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:19:	AuditCSV	90.2%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:89:	AuditRows	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:101:	mapHeader	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:148:	canonicalHeader	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:155:	rawFrom		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:173:	trimRaw		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:183:	isFieldCount	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/parse.go:188:	lineOf		75.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:23:	audit		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:78:	validate	97.1%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:145:	toRow		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:163:	parseDate	87.5%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:178:	mustDate	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:183:	validMonth	90.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:200:	plainDecimal	84.6%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:221:	parseCoverage	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:232:	hiredAfterMonth	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:241:	controlReason	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:250:	duplicateReason	87.5%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:267:	sortExceptions	58.3%
github.com/isaacbrendel/employee-hours-audit/internal/record/validate.go:288:	summarize	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:24:	Write		75.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:38:	Build		60.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:47:	populate	62.5%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:93:	makeStyles	75.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:134:	writeClean	68.2%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:222:	markFullTime	70.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:247:	writeExceptions	71.4%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:298:	writeSummary	60.9%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:385:	aggFormula	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:400:	writeMonthChart	79.3%
github.com/isaacbrendel/employee-hours-audit/internal/report/report.go:456:	writePivot	80.0%
github.com/isaacbrendel/employee-hours-audit/internal/report/safe.go:9:		SafeText	100.0%
github.com/isaacbrendel/employee-hours-audit/internal/suggest/anthropic.go:35:	FromEnv		42.9%
github.com/isaacbrendel/employee-hours-audit/internal/suggest/anthropic.go:52:	Suggest		76.5%
github.com/isaacbrendel/employee-hours-audit/internal/suggest/anthropic.go:135:	prompt		100.0%
github.com/isaacbrendel/employee-hours-audit/internal/suggest/anthropic.go:149:	rawValue	25.0%
github.com/isaacbrendel/employee-hours-audit/internal/suggest/anthropic.go:168:	parseSuggestion	86.7%
total:										(statements)	79.1%
```

## FuzzParse, 60 seconds

```
fuzz: elapsed: 0s, gathering baseline coverage: 0/5 completed
fuzz: elapsed: 0s, gathering baseline coverage: 5/5 completed, now fuzzing with 8 workers
fuzz: elapsed: 3s, execs: 52010 (17334/sec), new interesting: 36 (total: 41)
fuzz: elapsed: 6s, execs: 63199 (3730/sec), new interesting: 45 (total: 50)
fuzz: elapsed: 9s, execs: 63199 (0/sec), new interesting: 45 (total: 50)
fuzz: elapsed: 12s, execs: 337066 (91322/sec), new interesting: 58 (total: 63)
fuzz: elapsed: 15s, execs: 354595 (5844/sec), new interesting: 59 (total: 64)
fuzz: elapsed: 18s, execs: 487704 (44362/sec), new interesting: 60 (total: 65)
fuzz: elapsed: 21s, execs: 487704 (0/sec), new interesting: 60 (total: 65)
fuzz: elapsed: 24s, execs: 616335 (42868/sec), new interesting: 63 (total: 68)
fuzz: elapsed: 27s, execs: 616335 (0/sec), new interesting: 63 (total: 68)
fuzz: elapsed: 30s, execs: 616335 (0/sec), new interesting: 63 (total: 68)
fuzz: elapsed: 33s, execs: 616335 (0/sec), new interesting: 63 (total: 68)
fuzz: elapsed: 36s, execs: 616335 (0/sec), new interesting: 63 (total: 68)
fuzz: elapsed: 39s, execs: 900984 (94882/sec), new interesting: 64 (total: 69)
fuzz: elapsed: 42s, execs: 1478337 (192420/sec), new interesting: 75 (total: 80)
fuzz: elapsed: 45s, execs: 1506435 (9366/sec), new interesting: 92 (total: 97)
fuzz: elapsed: 48s, execs: 1548539 (14037/sec), new interesting: 108 (total: 113)
fuzz: elapsed: 51s, execs: 1584528 (11995/sec), new interesting: 123 (total: 128)
fuzz: elapsed: 54s, execs: 1594844 (3438/sec), new interesting: 126 (total: 131)
fuzz: elapsed: 57s, execs: 1889053 (98099/sec), new interesting: 130 (total: 135)
fuzz: elapsed: 1m0s, execs: 1926987 (12645/sec), new interesting: 132 (total: 137)
fuzz: elapsed: 1m1s, execs: 1926987 (0/sec), new interesting: 132 (total: 137)
PASS
ok  	github.com/isaacbrendel/employee-hours-audit/internal/record	61.611s
```

## Vitest

Command: `npx vitest run` in `web/`.

```
 RUN  v4.1.11 /Users/isaac-brendel-creator/employee-hours-audit/web

 Test Files  1 passed (1)
      Tests  6 passed (6)
   Start at  14:02:08
   Duration  712ms (transform 148ms, setup 0ms, import 179ms, tests 20ms, environment 0ms)
```
