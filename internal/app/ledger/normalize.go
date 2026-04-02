package ledger

import "strings"

func normalizePartyName(raw string) string {
	party := strings.TrimSpace(raw)
	party = strings.Split(party, "*")[0]
	party = strings.Split(party, "\t")[0]
	party = strings.Join(strings.Fields(strings.TrimSpace(party)), " ")
	return party
}

func sanitizeAmount(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.TrimPrefix(value, "Rp")
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", ".")
	value = strings.ReplaceAll(value, "(", "-")
	value = strings.ReplaceAll(value, ")", "")
	return value
}
