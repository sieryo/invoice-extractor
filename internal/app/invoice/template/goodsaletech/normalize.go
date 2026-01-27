package goodsaletech

import (
	"regexp"
	"strings"
)

func (t *GoodSaleTechTemplate) Normalize(raw string) string {
	// 1. Normalisasi newline
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")
	var cleaned []string

	for _, line := range lines {
		// 2. Trim kiri kanan
		line = strings.TrimSpace(line)

		// 3. Skip baris kosong total
		if line == "" {
			continue
		}

		line = regexp.MustCompile(`\s{2,}`).ReplaceAllString(line, "  ")

		cleaned = append(cleaned, line)
	}

	// 5. Gabungkan kembali
	return strings.Join(cleaned, "\n")
}
