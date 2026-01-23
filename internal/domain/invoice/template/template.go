package template

import "github.com/sieryo/invoice-extractor/internal/domain/invoice"

type Template interface {
	Name() string
	Match(raw string) bool

	Normalize(raw string) (string, error)
	Parse(normalized string) (*invoice.Invoice, error)
}
