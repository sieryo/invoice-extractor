package helper

import "time"

func FormatDateDDMMYYYY(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("02/01/2006")
}

func FormatDateYYYYMMDD(t *time.Time) string {
	if t == nil || t.IsZero() {
		return ""
	}
	return t.Format("2006-01-02")
}
