# Messy hours fixture

Synthetic people. No real personal information.

`employees_messy.csv` is UTF-8 with a leading BOM, CRLF line endings, and two trailing blank lines. Blank lines are not records. `encoding/csv` skips them, and the audit does not invent an exception for them.

The header is `Employee_ID,NAME,Hire_Date,Month,Hours_Worked,Coverage_Offered,Department`. Casing and the space-form of those names are accepted. `Department` is an extra column. It produces one warning and is not copied into the clean rows.

Row numbers in the audit are physical line numbers. The header is line 1, so the first data row is line 2.

## Clean rows

These pass every rule. Full-time means 130 or more hours. A coverage gap is a flag on a clean row, not an exception.

| ID | What it plants |
| --- | --- |
| E001 | Straightforward full-time month with coverage |
| E002 | Full-time month, coverage `no`. Coverage gap |
| E003 | Quoted comma in the name. Coverage `N` |
| E004 | Hire date `03/15/2015`. 129.5 hours, so not full-time. Coverage `TRUE` |
| E005 | Hire date `15-Mar-2012`. Exactly 130 hours, full-time |
| E006 | Spaces around the name, hours, and coverage. Trimmed, then accepted |
| 00042 | Leading zeros in the employee id stay an id, not a number |
| E008 | 37.5 hours. Coverage `1` |
| E009 | Name `=1+1`. Valid data. The workbook must store it as text |
| E010 | Name `@SUM(1,1)`. Same rule |
| E011 | Name `+1+1`. Coverage `y` |
| E012 | Leap day `2024-02-29` |
| E013 | Zero hours, which is a reported value, not a missing value |
| E014 | 744 hours, the last value under the 31-day ceiling |
| E015 | Spaces around the employee id. Coverage `f` |
| E016 | Hire date `2020/01/02` |
| E017 | Hire date `1/2/2020` |
| E018 | Coverage `YES`, 150 hours, full-time |

## Rows held for review

| ID | What is wrong |
| --- | --- |
| (blank id, name Missing Id) | `employee_id` is empty |
| E020 | Name is empty |
| E021 | Hire date is empty |
| E022 | Month is empty |
| E023 | Hours are empty |
| E024 | Coverage is empty |
| E025 | Hire date `March 5th 2024` is not in the accepted list |
| E026 | `13/01/2024` is not read as 13 January. Slash dates are month/day/year |
| E027 | Month `2024/07` is not `YYYY-MM` |
| E028 | Month `2024-13` |
| E029 | Month `202407` |
| E030 | Hours `abc` |
| E031 | Hours `-4` |
| E032 | Hours `800`, above 744. The value is not capped |
| E033 | Hours `1,200` |
| E034 | Hours `1e2` |
| E035 | Coverage `maybe` |
| E036 | Coverage `offered` |
| E037 | Two rows for 2024-09. Both are held, including the first |
| E039 | Hired `2024-10-01` for the month `2024-09` |
| E040 | Five columns against a seven-column header. The row is not mapped |
| E041 | Hours are only spaces, which trim to empty |
| E042 | Name is only spaces |
| E043 | Hire date `02/31/2024` is not a calendar day |
| E044 | Hours `nope` and coverage `perhaps` on the same row |
| (all fields empty) | Six missing required fields, six exceptions |
| E046 | Name contains a tab |

Separate unit tests, not this file, cover a broken quote, a duplicate header, a missing header, a byte that is not UTF-8, a blank line between records, and a BOM written as raw bytes.

`employees_messy.golden.json` is the audit of this file. Regenerate it only after reading the diff: `UPDATE_GOLDEN=1 go test ./internal/record -run TestGolden`.
