package parserhelper

import (
	"regexp"
	"strings"

	"github.com/sieryo/invoice-extractor/pkg/helper"
)

func ExtractValueAndAddress(line string) (string, string) {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return "", ""
	}

	rest := strings.TrimSpace(line[idx+1:])
	if rest == "" {
		return "", ""
	}

	parts := regexp.MustCompile(`\s{2,}`).Split(rest, 2)

	val := strings.TrimSpace(parts[0])
	if val == "" {
		return "", ""
	}

	if len(parts) == 2 {
		return val, strings.TrimSpace(parts[1])
	}

	return val, ""
}

func ExtractValue(line string) string {
	idx := strings.Index(line, ":")
	if idx == -1 {
		return ""
	}
	return strings.TrimSpace(line[idx+1:])
}

func CleanString(raw string) string {
	return helper.CleanString(raw)
}

func CleanAddress(addr string) string {
	addr = regexp.MustCompile(`(?i)\baddress\s*:\s*`).ReplaceAllString(addr, "")
	addr = strings.TrimSpace(addr)
	addr = strings.Trim(addr, ",")
	addr = regexp.MustCompile(`\s{2,}`).ReplaceAllString(addr, " ")
	return addr
}

func IsTableHeader(line string) bool {
	return strings.HasPrefix(line, "No") &&
		strings.Contains(line, "SKU") &&
		strings.Contains(line, "QTY")
}

func matchGroup(
	re *regexp.Regexp,
	match []string,
	name string,
) string {

	if re == nil || match == nil {
		return ""
	}

	idx := re.SubexpIndex(name)
	if idx < 0 || idx >= len(match) {
		return ""
	}

	return strings.TrimSpace(match[idx])
}
