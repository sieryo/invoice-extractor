package ledger

import "time"

type SnapshotItem struct {
	ID      string    `json:"id"`
	Date    time.Time `json:"date"`
	Account string    `json:"account"`
	RefNo   string    `json:"refNo"`
	Amount  float64   `json:"amount"`
	Tax     string    `json:"tax"`
	Status  string    `json:"status"`
}

type SnapshotParty struct {
	Party string         `json:"party"`
	Items []SnapshotItem `json:"items"`
}

type Snapshot struct {
	PeriodStart string                    `json:"periodStart,omitempty"`
	PeriodEnd   string                    `json:"periodEnd,omitempty"`
	Parties     map[string]*SnapshotParty `json:"parties"`
}

type PreviewRow struct {
	Party     string  `json:"party"`
	ItemCount int     `json:"itemCount"`
	Total     float64 `json:"total"`
}

type Preview struct {
	PeriodStart string       `json:"periodStart,omitempty"`
	PeriodEnd   string       `json:"periodEnd,omitempty"`
	PartyCount  int          `json:"partyCount"`
	ItemCount   int          `json:"itemCount"`
	Rows        []PreviewRow `json:"rows,omitempty"`
}
