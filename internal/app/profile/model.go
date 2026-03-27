package profile

import "time"

type Profile struct {
	ID           string
	Name         string
	Alias        string
	CutoffDate   int
	NPWP         string
	TKUID        string
	PasswordHash string
	CreatedAt    time.Time
}

