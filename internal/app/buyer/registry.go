package buyer

import (
	"strings"
	"sync"

	domainbuyer "github.com/sieryo/invoice-extractor/internal/domain/buyer"
)

type Registry struct {
	mu     sync.RWMutex
	loaded bool
	byName map[string]domainbuyer.Buyer
}

func NewRegistry() *Registry {
	return &Registry{
		loaded: false,
		byName: make(map[string]domainbuyer.Buyer),
	}
}

func normalizeName(name string) string {
	blacklist := []string{
		"perseroan terbatas",
		"perseroan",
		"terbatas",
		"badan",
		"pt.",
		"cv.",
		"pt",
		"cv",
		"-",
		".",
	}

	s := strings.ToLower(name)

	for _, w := range blacklist {
		s = strings.ReplaceAll(s, w, "")
	}

	return strings.Join(strings.Fields(s), " ")
}

func (r *Registry) IsLoaded() bool {
	return r.loaded
}

func (r *Registry) Load(buyers []domainbuyer.Buyer) {
	r.mu.Lock()
	defer r.mu.Unlock()

	m := make(map[string]domainbuyer.Buyer)
	for _, b := range buyers {
		m[normalizeName(b.Name)] = b
	}

	r.byName = m
	r.loaded = true
}

func (r *Registry) GetByName(name string) (domainbuyer.Buyer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	b, ok := r.byName[normalizeName(name)]

	return b, ok
}
