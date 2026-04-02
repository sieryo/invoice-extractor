package ledger

import (
	"sort"
	"strconv"
)

func parseSnapshotAmount(raw string) (float64, bool) {
	value := sanitizeAmount(raw)
	if value == "" {
		return 0, false
	}
	n, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0, false
	}
	return n, true
}

func BuildPreview(snapshot *Snapshot, limit int) Preview {
	if snapshot == nil {
		return Preview{}
	}
	if limit <= 0 {
		limit = 5
	}

	out := Preview{
		PeriodStart: snapshot.PeriodStart,
		PeriodEnd:   snapshot.PeriodEnd,
		PartyCount:  len(snapshot.Parties),
		Rows:        make([]PreviewRow, 0, limit),
	}

	parties := make([]*SnapshotParty, 0, len(snapshot.Parties))
	for _, party := range snapshot.Parties {
		parties = append(parties, party)
	}
	sort.Slice(parties, func(i, j int) bool {
		return parties[i].Party < parties[j].Party
	})

	for _, party := range parties {
		row := PreviewRow{
			Party:     party.Party,
			ItemCount: len(party.Items),
		}
		for _, item := range party.Items {
			row.Total += item.Amount
			out.ItemCount++
		}
		if len(out.Rows) < limit {
			out.Rows = append(out.Rows, row)
		}
	}

	return out
}
