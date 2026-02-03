package template

type TemplateRegistryService struct {
	registry *Registry
}

func NewTemplateRegistryService(registry *Registry) *TemplateRegistryService {
	return &TemplateRegistryService{
		registry: registry,
	}
}

func (s *TemplateRegistryService) List() []TemplateInfo {
	templates := make([]TemplateInfo, 0, len(s.registry.list))
	for _, t := range s.registry.list {
		templates = append(templates, TemplateInfo{
			Identifier: t.Identifier(),
			Name:       t.Name(),
			Alias:      t.Alias(),
		})
	}
	return templates
}
