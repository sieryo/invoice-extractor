package bukpot

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/sieryo/invoice-extractor/internal/domain/bukpot"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
)

var ErrNoParserMatched = errors.New("no parser matched")

type Service struct {
	extractor TextExtractor

	mu      sync.RWMutex
	parsers []bukpot.Parser
	byKind  map[bukpot.Kind]bukpot.Parser
}

type TextExtractor interface {
	ExtractText(ctx context.Context, pdfPath string, opts pdftool.ExtractOptions) (string, error)
}

func NewService(extractor TextExtractor) *Service {
	return &Service{
		extractor: extractor,
		parsers:   make([]bukpot.Parser, 0),
		byKind:    make(map[bukpot.Kind]bukpot.Parser),
	}
}

func (s *Service) RegisterParser(parser bukpot.Parser) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !parser.Kind().IsValid() {
		return fmt.Errorf("invalid parser kind: %s", parser.Kind())
	}

	if _, exists := s.byKind[parser.Kind()]; exists {
		return fmt.Errorf("parser already registered for kind: %s", parser.Kind())
	}

	s.parsers = append(s.parsers, parser)
	s.byKind[parser.Kind()] = parser
	return nil
}

func (s *Service) ParseFile(
	ctx context.Context,
	input bukpot.FileInput,
	forcedKind *bukpot.Kind,
) (*bukpot.ParsedFile, error) {
	text, err := s.extractor.ExtractText(ctx, input.Path, pdftool.DefaultOptions())
	if err != nil {
		msg := err.Error()
		return &bukpot.ParsedFile{
			Input: input,
			Error: &msg,
		}, nil
	}

	parser, err := s.pickParser(text, forcedKind)
	if err != nil {
		msg := err.Error()
		return &bukpot.ParsedFile{
			Input: input,
			Error: &msg,
		}, nil
	}

	doc, err := parser.Parse(ctx, text)
	if err != nil {
		msg := err.Error()
		return &bukpot.ParsedFile{
			Input: input,
			Error: &msg,
		}, nil
	}

	return &bukpot.ParsedFile{
		Input: input,
		Data:  doc,
	}, nil
}

func (s *Service) pickParser(
	text string,
	forcedKind *bukpot.Kind,
) (bukpot.Parser, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if forcedKind != nil {
		if !forcedKind.IsValid() {
			return nil, fmt.Errorf("invalid forced kind: %s", *forcedKind)
		}
		p, ok := s.byKind[*forcedKind]
		if !ok {
			return nil, fmt.Errorf("parser for kind %s not registered", *forcedKind)
		}
		return p, nil
	}

	for _, p := range s.parsers {
		if p.Match(text) {
			return p, nil
		}
	}

	return nil, ErrNoParserMatched
}
