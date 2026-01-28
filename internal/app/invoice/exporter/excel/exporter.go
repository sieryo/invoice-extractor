package excel

import (
	"bytes"
	"context"

	"github.com/xuri/excelize/v2"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
)

type ExcelExporter struct {
	templateRegistry *template.Registry
	template         []byte
}

func NewExcelExporter(templateRegistry *template.Registry) *ExcelExporter {
	return &ExcelExporter{
		templateRegistry: templateRegistry,
		template:         invoiceTemplate,
	}
}

func (e *ExcelExporter) Export(
	ctx context.Context,
	invoices []*invoice.Invoice,
) ([]byte, error) {

	f, err := excelize.OpenReader(bytes.NewReader(e.template))
	if err != nil {
		return nil, err
	}

	// isi excel
	if err := writeInvoices(f, invoices, e.templateRegistry); err != nil {
		return nil, err
	}

	buf, err := f.WriteToBuffer()
	if err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}
