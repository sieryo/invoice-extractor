package rename

import (
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
)

type RenamedFile struct {
	ID         string
	Name       string
	SourceID   string
	SourceName string
	SourceURI  string
}

type BatchRenameResult struct {
	Files  []RenamedFile
	Errors []shared.FileResultError
	Audits []TaxInvoiceAudit
}
