package invoice

type Template interface {
	Name() string
	Match(raw string) bool

	Normalize(raw string) (string, error)
	Parse(normalized string) (*Invoice, error)
}
