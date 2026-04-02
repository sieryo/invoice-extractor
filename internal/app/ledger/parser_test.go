package ledger

import (
	"bufio"
	"strings"
	"testing"
)

func TestParseSnapshotSalesDetailPreview(t *testing.T) {
	raw := `PT. SEA MAKEUP BEAUTY
MPR Blok III No.13 RT.003 RW.011 Cilandak Barat, Jakarta Selatan

Sales [Customer Detail]

01/12/2025  through  31/12/2031
20/01/2026	Page 1
10.32.22
	ID#	Date	Quantity	Item/Acct	Description	Amount	Tax	Status

ALBACAR MEKAR NUSANTARA	*None
	SJ004600	15/12/2025		4-4100	Invoice No : SMB202512154231	195.835.657,00	PPN	Open
	SJ004601	15/12/2025		4-4100	Invoice No : SMB202512154230	100.255.135,00	PPN	Open
	SJ004602	15/12/2025		4-4100	Invoice No : SMB202512154092	58.049.722,00	PPN	Open

					ALBACAR MEKAR NUSANTARA Total:	569.578.346,00

ANEKA KOSMETIK	*None
	SJ004612	03/12/2025		4-4100	Invoice No : SMB202512033748	8.164.324,00	PPN	Open
	SJ004613	03/12/2025		4-4100	Invoice No : SMB202512033752	8.095.136,00	PPN	Open
	SJ004610	09/12/2025		4-4100	Invoice No : SMB202512093823	52.687.564,00	PPN	Open

					ANEKA KOSMETIK Total:	68.947.024,00

					Grand Total:	638.525.370,00`

	snapshot, err := ParseSnapshot(bufio.NewReader(strings.NewReader(raw)))
	if err != nil {
		t.Fatalf("parse snapshot: %v", err)
	}

	if snapshot.PeriodStart != "01/12/2025" || snapshot.PeriodEnd != "31/12/2031" {
		t.Fatalf("unexpected period: %s - %s", snapshot.PeriodStart, snapshot.PeriodEnd)
	}

	if len(snapshot.Parties) != 2 {
		t.Fatalf("unexpected party count: %d", len(snapshot.Parties))
	}

	albacar := snapshot.Parties["ALBACAR MEKAR NUSANTARA"]
	if albacar == nil {
		t.Fatalf("missing ALBACAR party")
	}
	if len(albacar.Items) != 3 {
		t.Fatalf("unexpected ALBACAR items: %d", len(albacar.Items))
	}
	if albacar.Items[0].Amount != 195835657 {
		t.Fatalf("unexpected ALBACAR first amount: %.2f", albacar.Items[0].Amount)
	}

	preview := BuildPreview(snapshot, 8)
	if preview.PartyCount != 2 {
		t.Fatalf("unexpected preview party count: %d", preview.PartyCount)
	}
	if preview.ItemCount != 6 {
		t.Fatalf("unexpected preview item count: %d", preview.ItemCount)
	}
	if len(preview.Rows) != 2 {
		t.Fatalf("unexpected preview rows: %d", len(preview.Rows))
	}

	if preview.Rows[0].Party != "ALBACAR MEKAR NUSANTARA" {
		t.Fatalf("unexpected first party: %s", preview.Rows[0].Party)
	}
	if preview.Rows[0].ItemCount != 3 {
		t.Fatalf("unexpected first party item count: %d", preview.Rows[0].ItemCount)
	}
	if preview.Rows[0].Total != 354140514 {
		t.Fatalf("unexpected first party total: %.2f", preview.Rows[0].Total)
	}
}
