package extract

import (
	"context"
	"fmt"
	"os"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"regexp"
	"strconv"

	"github.com/sieryo/invoice-extractor/internal/app/buyer"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/parserhelper"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/template"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/sieryo/invoice-extractor/internal/domain/shared"
	"github.com/sieryo/invoice-extractor/internal/infra/adapter/pdftool"
	"github.com/sieryo/invoice-extractor/internal/infra/jobrunner"
	"github.com/sieryo/invoice-extractor/pkg/helper"
)

var leadingNumberRegex = regexp.MustCompile(`^\s*(\d+)[\.\s]+`)

func extractLeadingNumber(filename string) (int, bool) {
	matches := leadingNumberRegex.FindStringSubmatch(filename)
	if len(matches) < 2 {
		return 0, false
	}

	n, err := strconv.Atoi(matches[1])
	if err != nil {
		return 0, false
	}

	return n, true
}

type InvoiceExtractorService struct {
	buyerRegistry    *buyer.Registry
	templateRegistry *template.Registry
}

func NewInvoiceExtractService(t *template.Registry, b *buyer.Registry) *InvoiceExtractorService {
	return &InvoiceExtractorService{
		buyerRegistry:    b,
		templateRegistry: t,
	}
}

func (i *InvoiceExtractorService) ExtractBatch(
	ctx context.Context,
	inputFiles []file.ResolvedFile,
	templateID *string,
) (*BatchExtractResult, error) {

	var wg sync.WaitGroup

	errChan := make(chan shared.FileResultError, len(inputFiles))
	invoiceChan := make(chan *invoice.Invoice, len(inputFiles))
	auditChan := make(chan InvoiceAudit, len(inputFiles))

	reporter := jobrunner.GetProgressReporter(ctx)

	var extracted int32
	total := int32(len(inputFiles))

	for _, f := range inputFiles {
		inputFile := f
		p := inputFile.Path

		wg.Add(1)
		go func(refFile file.ResolvedFile) {
			defer wg.Done()

			defer func() {
				current := atomic.AddInt32(&extracted, 1)

				if reporter != nil {
					p := float64(current) / float64(total)
					progress := int(p * 70)
					reporter.Report(progress)
				}
			}()

			// context turunan, bukan overwrite
			ctx2 := ctx
			if reporter != nil {
				ctx2 = jobrunner.WithProgressReporter(ctx, reporter)
			}

			if _, err := os.Stat(p); err != nil {
				errChan <- shared.FileResultError{FileID: refFile.ID, FileName: refFile.Name, Error: err.Error()}
				return
			}

			text, err := pdftool.ExtractText(ctx2, p, pdftool.DefaultOptions())
			if err != nil {
				errChan <- shared.FileResultError{FileID: refFile.ID, FileName: refFile.Name, Error: err.Error()}
				return
			}

			var tpl template.Template
			if templateID != nil {
				t, ok := i.templateRegistry.GetByIdentifier(*templateID)
				if !ok {
					errChan <- shared.FileResultError{
						FileID:   refFile.ID,
						FileName: refFile.Name,
						Error:    fmt.Sprintf("unknown template: %s", *templateID),
					}
					return
				}
				tpl = t
			} else {
				t, err := i.templateRegistry.Detect(text)
				if err != nil {
					errChan <- shared.FileResultError{
						FileID:   refFile.ID,
						FileName: refFile.Name,
						Error:    "no template matched",
					}
					return
				}
				tpl = t
			}

			normalized := tpl.Normalize(text)

			inv, err := tpl.Parse(text)
			if err != nil {
				errChan <- shared.FileResultError{
					FileID:   refFile.ID,
					FileName: refFile.Name,
					Error:    err.Error(),
				}
				return
			}

			// Inject metadata
			inv.Metadata = &invoice.InvoiceMetadata{
				SourceFile: invoice.FileRef{
					ID:   refFile.ID,
					Name: refFile.Name,
				},
				TemplateID:  tpl.Identifier(),
				ExtractedAt: time.Now(),
			}
			inv.Metadata.Warnings = parserhelper.BuildParseWarnings(inv)

			audit := InvoiceAudit{
				SourceFile: invoice.FileRef{
					ID:   refFile.ID,
					Name: refFile.Name,
				},
				TemplateID:     tpl.Identifier(),
				TemplateName:   tpl.Name(),
				NormalizedText: normalized,
				ExtractedAt:    inv.Metadata.ExtractedAt,
			}

			if inv.Buyer != nil {
				audit.Buyer.ParsedName = inv.Buyer.Name
				if inv.Buyer.TaxID != nil {
					audit.Buyer.ParsedTaxID = *inv.Buyer.TaxID
				}
				if inv.Buyer.TKU != nil {
					audit.Buyer.ParsedTKU = *inv.Buyer.TKU
				}
			}

			// buyer enrichment
			if inv.Buyer != nil && inv.Buyer.Name != "" {
				if b, ok := i.buyerRegistry.GetByName(inv.Buyer.Name); ok {
					rawTaxID := b.PrimaryTaxID()
					rawTKU := b.TKU()

					taxID := helper.DigitsOnly(rawTaxID)
					tku := helper.DigitsOnly(rawTKU)

					taxIDKind := helper.TaxIDKind(taxID)
					taxIDValid := helper.IsValidTaxID(taxID)
					tkuValid := helper.IsNITKU(tku)

					inv.Buyer.Name = b.Name
					if taxIDValid {
						inv.Buyer.TaxID = &taxID
					} else if rawTaxID != "" {
						inv.Metadata.Warnings = append(inv.Metadata.Warnings, "invalid buyer tax id in registry")
					}

					if tkuValid {
						inv.Buyer.TKU = &tku
					} else if rawTKU != "" {
						inv.Metadata.Warnings = append(inv.Metadata.Warnings, "invalid buyer tku in registry")
					}

					audit.Buyer.Enriched = true
					audit.Buyer.RegistryName = b.Name
					audit.Buyer.RegistryTaxID = rawTaxID
					audit.Buyer.RegistryTKU = rawTKU
					audit.Buyer.AppliedTaxID = taxID
					audit.Buyer.AppliedTKU = tku
					audit.Buyer.TaxIDKind = taxIDKind
					audit.Buyer.TaxIDValid = taxIDValid
					audit.Buyer.TKUValid = tkuValid
				}
			}

			applyTestWarningHooks(inv, refFile.Name)

			audit.Warnings = inv.Metadata.Warnings

			invoiceChan <- inv
			auditChan <- audit

		}(inputFile)
	}

	go func() {
		wg.Wait()
		close(errChan)
		close(invoiceChan)
		close(auditChan)
	}()

	result := &BatchExtractResult{}

	invoiceDone := false
	errorDone := false
	auditDone := false

	for !(invoiceDone && errorDone && auditDone) {
		select {
		case err, ok := <-errChan:
			if !ok {
				errorDone = true
				continue
			}
			result.Errors = append(result.Errors, err)

		case inv, ok := <-invoiceChan:
			if !ok {
				invoiceDone = true
				continue
			}
			result.Invoices = append(result.Invoices, inv)

		case audit, ok := <-auditChan:
			if !ok {
				auditDone = true
				continue
			}
			result.Audits = append(result.Audits, audit)

		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}

	sort.Slice(result.Invoices, func(i, j int) bool {
		fi := result.Invoices[i].Metadata.SourceFile.Name
		fj := result.Invoices[j].Metadata.SourceFile.Name

		ni, okI := extractLeadingNumber(fi)
		nj, okJ := extractLeadingNumber(fj)

		// Dua-duanya punya angka → bandingkan angka
		if okI && okJ {
			return ni < nj
		}

		// Hanya i yang punya angka → i duluan
		if okI && !okJ {
			return true
		}

		// Hanya j yang punya angka → j duluan
		if !okI && okJ {
			return false
		}

		// Dua-duanya gak punya angka → biarin stabil
		return false
	})

	sort.Slice(result.Audits, func(i, j int) bool {
		fi := result.Audits[i].SourceFile.Name
		fj := result.Audits[j].SourceFile.Name

		ni, okI := extractLeadingNumber(fi)
		nj, okJ := extractLeadingNumber(fj)

		if okI && okJ {
			return ni < nj
		}
		if okI && !okJ {
			return true
		}
		if !okI && okJ {
			return false
		}
		return false
	})

	if reporter != nil {
		total := len(result.Invoices)
		for idx := range result.Invoices {
			progress := int(float64(idx+1) / float64(total) * 100)
			reporter.Report(progress)
		}
	}

	return result, nil
}

func applyTestWarningHooks(inv *invoice.Invoice, sourceFileName string) {
	if inv == nil || inv.Metadata == nil {
		return
	}

	// Force a warning without changing the parsed invoice payload.
	if isTruthyEnv("INVOICE_TEST_FORCE_WARNING") {
		appendWarning(inv, "forced warning (test mode)")
	}

	// Force warning only for selected files (substring match, case-insensitive).
	pattern := strings.TrimSpace(os.Getenv("INVOICE_TEST_FORCE_WARNING_FILE_CONTAINS"))
	if pattern != "" && strings.Contains(strings.ToLower(sourceFileName), strings.ToLower(pattern)) {
		appendWarning(inv, fmt.Sprintf("forced warning for file match: %s", pattern))
	}

	// Optional test mode: drop parsed items to simulate incomplete extraction.
	// This changes output intentionally and should only be used in test/local.
	if isTruthyEnv("INVOICE_TEST_DROP_ITEMS") {
		inv.Items = nil
		appendWarning(inv, "no items parsed (forced test mode)")
	}
}

func appendWarning(inv *invoice.Invoice, warning string) {
	if inv == nil || inv.Metadata == nil || warning == "" {
		return
	}
	for _, existing := range inv.Metadata.Warnings {
		if existing == warning {
			return
		}
	}
	inv.Metadata.Warnings = append(inv.Metadata.Warnings, warning)
}

func isTruthyEnv(key string) bool {
	value := strings.TrimSpace(strings.ToLower(os.Getenv(key)))
	switch value {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}
