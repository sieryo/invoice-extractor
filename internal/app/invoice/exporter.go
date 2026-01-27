package invoice

import "context"

type ExportStat struct {
	Total   int
	Success int
	Failed  int
}

type InvoiceExporter interface {
	Export(ctx context.Context, invoices []*Invoice) ([]byte, error)
}
