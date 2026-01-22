package services

import "github.com/sieryo/invoice-extractor/internal/invoice"

// Return akhirnya adalah: struct Invoice with items atau nil (jika error). Ini harusnya singleton ya?
type InvoiceExtractorService struct {
}

func NewInvoiceExtractor() *InvoiceExtractorService {
	return &InvoiceExtractorService{}
}

func (*InvoiceExtractorService) Extract(template invoice.Template) {

	// Ini extractor butuh paths dari pdf yang akan di-proses. Jadi yang convert filenya ke temp bukanlah service ini, melainkan service lain.
	// Semua service di-init pertama kali saat server dijalankan? Service nyentuh repo itu wajar keknya?
	// Intinya kalau gagal, handler gak tau apa-apa dan return error aja. Sedangkan service harus cleanup semua yang dilakukan.
}
