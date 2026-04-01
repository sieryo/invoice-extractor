package cashflowbill

import (
	"strings"
	"testing"
	"time"
)

func TestBuildReceivePaymentsRowsAddsBlankLinePerItem(t *testing.T) {
	tx := Transaction{
		Party: "ANEKA KOSMETIK",
		Items: []TransactionItem{
			{
				RefID:      "SJ1",
				RefDate:    time.Date(2025, 12, 3, 0, 0, 0, 0, time.UTC),
				ActionDate: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
				Amount:     1000,
				Memo:       "memo-1",
			},
			{
				RefID:      "SJ2",
				RefDate:    time.Date(2025, 12, 4, 0, 0, 0, 0, time.UTC),
				ActionDate: time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC),
				Amount:     2000,
				Memo:       "memo-2",
			},
		},
	}

	rows := BuildReceivePaymentsRows(tx, "12021")
	if len(rows) != 4 {
		t.Fatalf("unexpected row count: got %d want 4", len(rows))
	}
	if len(rows[1]) != 0 || len(rows[3]) != 0 {
		t.Fatalf("expected blank separator row after each item")
	}

	body, err := EncodeTabDelimitedText(append([][]string{ReceivePaymentsHeader()}, rows...))
	if err != nil {
		t.Fatalf("encode text: %v", err)
	}

	text := string(body)
	if !strings.Contains(text, "\n\nANEKA KOSMETIK") {
		t.Fatalf("expected blank line between entries, got:\n%s", text)
	}
}

func TestEncodeTabDelimitedTextSupportsDoubleBlankLineBetweenGroups(t *testing.T) {
	body, err := EncodeTabDelimitedText([][]string{
		ReceivePaymentsHeader(),
		{"PARTY A", "", "12021"},
		{},
		{},
		{"PARTY B", "", "12021"},
		{},
	})
	if err != nil {
		t.Fatalf("encode text: %v", err)
	}

	text := string(body)
	if !strings.Contains(text, "PARTY A\t\t12021\n\n\nPARTY B\t\t12021") {
		t.Fatalf("expected double blank line between groups, got:\n%s", text)
	}
}

func TestBuildReceivePaymentsRowsClearsMemoForCreditRow(t *testing.T) {
	tx := Transaction{
		Party: "BEAUTY PRATAMA",
		Items: []TransactionItem{
			{
				RefID:      "",
				RefDate:    time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
				ActionDate: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
				Amount:     114255180,
				Memo:       "BEAUTY PRATAMA CV",
			},
		},
	}

	rows := BuildReceivePaymentsRows(tx, "12021")
	if len(rows) == 0 {
		t.Fatalf("expected rows")
	}
	if got := rows[0][9]; got != "" {
		t.Fatalf("expected blank memo for credit row, got %q", got)
	}
}

func TestBuildPayBillsRowsClearsMemoForCreditRow(t *testing.T) {
	tx := Transaction{
		Party: "SUPPLIER A",
		Items: []TransactionItem{
			{
				RefID:      "",
				RefDate:    time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
				ActionDate: time.Date(2025, 12, 10, 0, 0, 0, 0, time.UTC),
				Amount:     50000,
				Memo:       "memo source",
			},
		},
	}

	rows := BuildPayBillsRows(tx, "12021")
	if len(rows) == 0 {
		t.Fatalf("expected rows")
	}
	if got := rows[0][14]; got != "" {
		t.Fatalf("expected blank memo for credit row, got %q", got)
	}
}
