package document

import "fmt"

type ProcessorKey struct {
	CollectionKind CollectionKind
	SourceFormat   SourceFormat
}

type Registry struct {
	byKey map[ProcessorKey]DocumentProcessor
}

func NewRegistry() *Registry {
	return &Registry{
		byKey: make(map[ProcessorKey]DocumentProcessor),
	}
}

func (r *Registry) Register(p DocumentProcessor) error {
	if p == nil {
		return ErrNilProcessor
	}

	key := p.Key()
	if !key.CollectionKind.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidDocumentType, key.CollectionKind)
	}
	if !key.SourceFormat.IsValid() {
		return fmt.Errorf("%w: %s", ErrInvalidDocumentType, key.SourceFormat)
	}

	if _, exists := r.byKey[key]; exists {
		return fmt.Errorf("%w: %s/%s", ErrProcessorAlreadyRegistered, key.CollectionKind, key.SourceFormat)
	}

	r.byKey[key] = p
	return nil
}

func (r *Registry) MustRegister(p DocumentProcessor) {
	if err := r.Register(p); err != nil {
		panic(err)
	}
}

func (r *Registry) Get(key ProcessorKey) (DocumentProcessor, bool) {
	p, ok := r.byKey[key]
	return p, ok
}

func (r *Registry) Keys() []ProcessorKey {
	out := make([]ProcessorKey, 0, len(r.byKey))
	for key := range r.byKey {
		out = append(out, key)
	}
	return out
}
