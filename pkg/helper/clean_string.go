package helper

import "strings"

func CleanString(s string) string {
	if s == "" {
		return s
	}

	// ganti unicode escape umum
	replacer := strings.NewReplacer(
		"\u0026", "&",
		"\u00a0", " ", // non-breaking space
	)

	s = replacer.Replace(s)

	// newline, tab → spasi
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\t", " ")

	// rapihin spasi berlebih
	s = strings.Join(strings.Fields(s), " ")

	return strings.TrimSpace(s)
}
