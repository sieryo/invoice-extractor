package template

import "github.com/sieryo/invoice-extractor/internal/app/invoice"

type Template interface {
	Identifier() string
	Match(raw string) bool

	Normalize(raw string) string
	Parse(normalized string) (*invoice.Invoice, error)
}
