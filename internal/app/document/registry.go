package document

import "fmt"

type Registry struct {
	byType map[DocumentType]DocumentProcessor
}

func NewRegistry() *Registry {
	return &Registry{
		byType: make(map[DocumentType]DocumentProcessor),
	}
}

func (r *Registry) Register(p DocumentProcessor) error {
	if p == nil {
		return ErrNilProcessor
	}

	docType := p.Type()
	if !docType.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidDocumentType, docType)
	}

	if _, exists := r.byType[docType]; exists {
		return fmt.Errorf("%w: %s", ErrProcessorAlreadyRegistered, docType)
	}

	r.byType[docType] = p
	return nil
}

func (r *Registry) MustRegister(p DocumentProcessor) {
	if err := r.Register(p); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(docType DocumentType) (DocumentProcessor, bool) {
	p, ok := r.byType[docType]
	return p, ok
}

func (r *Registry) Types() []DocumentType {
	out := make([]DocumentType, 0, len(r.byType))
	for t := range r.byType {
		out = append(out, t)
	}
	return out
}
