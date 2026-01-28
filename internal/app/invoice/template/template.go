package template

import "github.com/sieryo/invoice-extractor/internal/app/invoice"

type Template interface {
	Identifier() string
	Name() string
	Match(raw string) bool

	Seller() *invoice.Party

	Normalize(raw string) string
	Parse(normalized string) (*invoice.Invoice, error)
}
