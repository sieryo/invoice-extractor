package seamakeup

import (
	"regexp"
	"strings"
)

var (
	itemStartRe  = regexp.MustCompile(`^\d+\s+\d{4,}`)
	numberLineRe = regexp.MustCompile(`\d+[,\.]\d+`)
)

func normalizeSeaMakeup(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")

	lines := strings.Split(raw, "\n")
	tmp := make([]string, 0, len(lines))

	// --- PASS 1: normalize header rows ---
	for i := 0; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		// gabung "(Excl. VAT)" ke baris sebelumnya
		if line == "(Excl. VAT)" && len(tmp) > 0 {
			tmp[len(tmp)-1] += " (Excl. VAT)"
			continue
		}

		tmp = append(tmp, line)
	}

	// --- PASS 2: normalize item rows ---
	out := make([]string, 0, len(tmp))

	var buf string
	inItems := false

	for _, line := range tmp {

		if strings.HasPrefix(line, "No  SKU") {
			inItems = true
			out = append(out, line)
			continue
		}

		if !inItems {
			out = append(out, line)
			continue
		}

		if itemStartRe.MatchString(line) {
			if buf != "" {
				out = append(out, buf)
			}
			buf = line
			continue
		}

		if buf != "" {
			buf += " " + line
			continue
		}

		out = append(out, line)
	}

	if buf != "" {
		out = append(out, buf)
	}

	return strings.Join(out, "\n")
}
