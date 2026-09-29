package report

import "unicode/utf8"

// SafeText prefixes a spreadsheet formula trigger with a single quote.
// OWASP's CSV Injection page lists = + - @ tab, CR, LF, and the full-width
// forms. This writes xlsx cells rather than a CSV file. The page also says
// no prefix is reliable for every spreadsheet program.
func SafeText(s string) string {
	if s == "" {
		return ""
	}
	r, _ := utf8.DecodeRuneInString(s)
	switch r {
	case '=', '+', '-', '@', '\t', '\r', '\n', '＝', '＋', '－', '＠':
		return "'" + s
	default:
		return s
	}
}
