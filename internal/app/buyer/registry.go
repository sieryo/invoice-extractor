package buyer

import (
	"strings"
	"sync"

	domainbuyer "github.com/sieryo/invoice-extractor/internal/domain/buyer"
)

type normalizedBuyer struct {
	Raw      domainbuyer.Buyer
	NormName string
	Tokens   []string
}

type matchScore struct {
	buyer      domainbuyer.Buyer
	matchCount int
	score      float64
}

type Registry struct {
	mu     sync.RWMutex
	loaded bool
	buyers []normalizedBuyer
}

func NewRegistry() *Registry {
	return &Registry{
		loaded: false,
		buyers: make([]normalizedBuyer, 0),
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

	norm := make([]normalizedBuyer, 0, len(buyers))

	for _, b := range buyers {
		n := normalizeName(b.Name)
		norm = append(norm, normalizedBuyer{
			Raw:      b,
			NormName: n,
			Tokens:   strings.Fields(n),
		})
	}

	r.buyers = norm
	r.loaded = true
}

func (r *Registry) GetByName(name string) (domainbuyer.Buyer, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if !r.loaded {
		return domainbuyer.Buyer{}, false
	}

	targetTokens := strings.Fields(normalizeName(name))

	var best *matchScore

	for _, nb := range r.buyers {
		ms := scoreBuyer(targetTokens, nb.Tokens, nb.Raw)
		if ms == nil {
			continue
		}

		if best == nil ||
			ms.matchCount > best.matchCount ||
			(ms.matchCount == best.matchCount && ms.score > best.score) {
			best = ms
		}
	}

	if best == nil {
		return domainbuyer.Buyer{}, false
	}

	return best.buyer, true
}

func tokenMatches(token string, namaTokens []string) bool {
	for _, nt := range namaTokens {
		if token == nt || strings.Contains(nt, token) {
			return true
		}
	}
	return false
}

func scoreBuyer(
	targetTokens []string,
	namaTokens []string,
	buyer domainbuyer.Buyer,
) *matchScore {

	matchCount := 0
	for _, t := range targetTokens {
		if tokenMatches(t, namaTokens) {
			matchCount++
		}
	}

	targetLen := len(targetTokens)
	namaLen := len(namaTokens)

	switch targetLen {
	case 1:
		if namaLen != 1 || matchCount != 1 {
			return nil
		}
	case 2:
		if matchCount != 2 {
			return nil
		}
	case 3:
		if matchCount < 2 {
			return nil
		}
	default:
		if matchCount < 3 {
			return nil
		}
	}

	return &matchScore{
		buyer:      buyer,
		matchCount: matchCount,
		score:      float64(matchCount) / float64(targetLen),
	}
}
