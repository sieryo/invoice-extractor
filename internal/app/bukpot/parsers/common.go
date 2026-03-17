package parsers

import (
	"regexp"
	"strings"
	"time"
)

var whitespaceRe = regexp.MustCompile(`\s+`)
var bukpotNomorRe = regexp.MustCompile(`\b[0-9A-Z]{8,}\b`)
var bukpotMasaRe = regexp.MustCompile(`\b\d{2}-\d{4}(?:-\d{2}-\d{4})?\b`)
var objectCodeRe = regexp.MustCompile(`\b\d{2}-\d{3}-\d{2}\b`)
var numericTokenRe = regexp.MustCompile(`^[\d.,]+$`)
var npwpNikRe = regexp.MustCompile(`\b\d{15,22}\b`)
var indoDateTextRe = regexp.MustCompile(`(?i)\b\d{1,2}\s+(januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember)\s+\d{4}\b`)
var tanggalTailRe = regexp.MustCompile(`(?i)\btanggal\b.*$`)

type topFields struct {
	Nomor           string
	Masa            string
	SifatPemotongan string
	StatusBukti     string
}

func normalizeText(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")
	cleaned := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = whitespaceRe.ReplaceAllString(line, " ")
		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}

func firstLineAfterPrefix(lines []string, prefix string) string {
	prefixLower := strings.ToLower(prefix)
	for _, line := range lines {
		if strings.HasPrefix(strings.ToLower(line), prefixLower) {
			if idx := strings.Index(line, ":"); idx >= 0 && idx+1 < len(line) {
				return strings.TrimSpace(line[idx+1:])
			}
			return strings.TrimSpace(line[len(prefix):])
		}
	}
	return ""
}

func captureUntil(lines []string, untilContains string) []string {
	if untilContains == "" {
		return lines
	}

	out := make([]string, 0, len(lines))
	for _, line := range lines {
		if strings.Contains(line, untilContains) {
			break
		}
		out = append(out, line)
	}
	return out
}

func parseTopFields(lines []string, stopAt string) topFields {
	area := captureUntil(lines, stopAt)
	var out topFields

	anchorIdx := -1
	for i, line := range area {
		if extractBukpotNumber(line) != "" && bukpotMasaRe.FindString(line) != "" {
			anchorIdx = i
			break
		}
	}

	// Fallback: masa pajak line is often split from nomor by one line in some renders.
	if anchorIdx < 0 {
		for i := 0; i < len(area); i++ {
			if bukpotMasaRe.FindString(area[i]) == "" {
				continue
			}
			start := max(0, i-1)
			end := min(len(area)-1, i+1)
			for j := start; j <= end; j++ {
				if extractBukpotNumber(area[j]) != "" {
					anchorIdx = i
					break
				}
			}
			if anchorIdx >= 0 {
				break
			}
		}
	}

	if anchorIdx < 0 {
		return out
	}

	windowStart := max(0, anchorIdx-1)
	windowEnd := min(len(area)-1, anchorIdx+3)

	for i := windowStart; i <= windowEnd; i++ {
		line := area[i]

		if out.Nomor == "" {
			out.Nomor = extractBukpotNumber(line)
		}
		if out.Masa == "" {
			out.Masa = bukpotMasaRe.FindString(line)
		}

		sifat, status := extractSifatStatus(line)
		if sifat == "TIDAK FINAL" {
			out.SifatPemotongan = sifat
		} else if out.SifatPemotongan == "" && sifat != "" {
			out.SifatPemotongan = sifat
		}
		if out.StatusBukti == "" && status != "" {
			out.StatusBukti = status
		}
	}

	return out
}

func extractBukpotNumber(line string) string {
	for _, token := range bukpotNomorRe.FindAllString(strings.ToUpper(line), -1) {
		if looksLikeBukpotNumber(token) {
			return token
		}
	}
	return ""
}

func extractSifatStatus(line string) (string, string) {
	upper := strings.ToUpper(line)

	sifat := ""
	if strings.Contains(upper, "TIDAK FINAL") {
		sifat = "TIDAK FINAL"
	} else if strings.Contains(upper, " FINAL") {
		sifat = "FINAL"
	}

	status := ""
	switch {
	case strings.Contains(upper, "DIBATALKAN"):
		status = "DIBATALKAN"
	case strings.Contains(upper, "PEMBETULAN"):
		status = "PEMBETULAN"
	case strings.Contains(upper, "PENGGANTI"):
		status = "PENGGANTI"
	case strings.Contains(upper, "NORMAL"):
		status = "NORMAL"
	}

	return sifat, status
}

func looksLikeBukpotNumber(token string) bool {
	hasDigit := false
	hasLetter := false
	for _, ch := range token {
		if ch >= '0' && ch <= '9' {
			hasDigit = true
		}
		if ch >= 'A' && ch <= 'Z' {
			hasLetter = true
		}
	}
	return hasDigit && hasLetter
}

func extractNPWPNIKAndName(raw string) (string, string) {
	line := strings.TrimSpace(raw)
	if line == "" {
		return "", ""
	}

	npwp := npwpNikRe.FindString(line)
	name := ""
	if idx := strings.Index(line, " - "); idx >= 0 {
		name = strings.TrimSpace(line[idx+3:])
	}

	return npwp, name
}

func parseIndonesianDate(raw string) string {
	s := strings.TrimSpace(raw)
	if s == "" {
		return ""
	}

	s = strings.ReplaceAll(s, ",", " ")
	s = strings.Join(strings.Fields(s), " ")

	layouts := []string{
		"02/01/2006",
		"2/1/2006",
		"02-01-2006",
		"2-1-2006",
		"2006-01-02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t.Format("02/01/2006")
		}
	}

	months := map[string]string{
		"januari": "01", "februari": "02", "maret": "03", "april": "04",
		"mei": "05", "juni": "06", "juli": "07", "agustus": "08",
		"september": "09", "oktober": "10", "november": "11", "desember": "12",
	}

	lower := strings.ToLower(s)
	parts := strings.Fields(lower)
	if len(parts) >= 3 {
		day := parts[0]
		month, ok := months[parts[1]]
		year := parts[2]
		if ok {
			if len(day) == 1 {
				day = "0" + day
			}
			return day + "/" + month + "/" + year
		}
	}

	return ""
}

func firstDateInText(raw string) string {
	matched := indoDateTextRe.FindString(raw)
	if matched == "" {
		return ""
	}
	return parseIndonesianDate(matched)
}

func normalizeDokumenReferensiJenis(raw string) string {
	val := strings.TrimSpace(raw)
	if val == "" {
		return ""
	}

	val = tanggalTailRe.ReplaceAllString(val, "")
	val = strings.TrimSpace(val)
	val = strings.TrimRight(val, ":")
	val = strings.TrimSpace(val)
	return val
}

func parseObjectFields(lines []string, startContains string, stopPrefixes ...string) (string, string) {
	start := -1
	for i, line := range lines {
		if strings.Contains(strings.ToUpper(line), strings.ToUpper(startContains)) {
			start = i + 1
			break
		}
	}
	if start < 0 {
		return "", ""
	}

	code := ""
	parts := make([]string, 0, 4)

	for i := start; i < len(lines); i++ {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		stop := false
		for _, prefix := range stopPrefixes {
			if strings.HasPrefix(strings.ToUpper(line), strings.ToUpper(prefix)) {
				stop = true
				break
			}
		}
		if stop {
			break
		}

		if code == "" {
			if token := objectCodeRe.FindString(line); token != "" {
				code = token
				line = strings.TrimSpace(strings.Replace(line, token, "", 1))
				line = trimTrailingNumericTokens(line)
				if line != "" {
					parts = append(parts, line)
				}
			}
			continue
		}

		line = trimTrailingNumericTokens(line)
		if line == "" || numericTokenRe.MatchString(line) {
			continue
		}
		parts = append(parts, line)
	}

	objek := strings.Join(parts, " ")
	objek = whitespaceRe.ReplaceAllString(strings.TrimSpace(objek), " ")
	return code, objek
}

func trimTrailingNumericTokens(raw string) string {
	tokens := strings.Fields(raw)
	if len(tokens) == 0 {
		return ""
	}

	end := len(tokens)
	for end > 0 && numericTokenRe.MatchString(tokens[end-1]) {
		end--
	}
	return strings.Join(tokens[:end], " ")
}

func deriveDocumentTag(candidates ...string) string {
	for _, raw := range candidates {
		value := strings.TrimSpace(raw)
		if value != "" {
			return value
		}
	}
	return ""
}
