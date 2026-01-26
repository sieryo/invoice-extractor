package template

import (
	"fmt"
)

type Registry struct {
	byID map[string]Template
	list []Template
}

func NewRegistry() *Registry {
	return &Registry{
		byID: make(map[string]Template),
		list: make([]Template, 0),
	}
}

func (r *Registry) Register(t Template) error {
	id := t.Identifier()

	if id == "" {
		return fmt.Errorf("template identifier cannot be empty")
	}

	if _, exists := r.byID[id]; exists {
		return fmt.Errorf("template with identifier %s already registered", id)
	}

	r.byID[id] = t
	r.list = append(r.list, t)
	return nil
}

func (r *Registry) GetByIdentifier(id string) (Template, bool) {
	t, ok := r.byID[id]
	return t, ok
}

func (r *Registry) Detect(raw string) (Template, error) {
	for _, t := range r.list {
		if t.Match(raw) {
			return t, nil
		}
	}
	return nil, fmt.Errorf("no template matched")
}
