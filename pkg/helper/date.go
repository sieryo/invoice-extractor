package helper

import (
	"fmt"
	"time"
)

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

func GetIndonesiaDateStr() string {
	now := time.Now()
	months := map[time.Month]string{
		time.January:   "Januari",
		time.February:  "Februari",
		time.March:     "Maret",
		time.April:     "April",
		time.May:       "Mei",
		time.June:      "Juni",
		time.July:      "Juli",
		time.August:    "Agustus",
		time.September: "September",
		time.October:   "Oktober",
		time.November:  "November",
		time.December:  "Desember",
	}
	dateStr := fmt.Sprintf("%02d %s %d", now.Day(), months[now.Month()], now.Year())

	return dateStr
}
