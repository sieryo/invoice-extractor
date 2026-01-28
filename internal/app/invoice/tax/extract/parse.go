package extract

import (
	"regexp"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
)

var refRe = regexp.MustCompile(
	`Referensi\s*:\s*([^)]+)`,
)

func normalize(raw string) string {
	raw = strings.ReplaceAll(raw, "\r\n", "\n")
	raw = strings.ReplaceAll(raw, "\r", "\n")

	lines := strings.Split(raw, "\n")
	var cleaned []string

	spaceRe := regexp.MustCompile(`\s{2,}`)

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		line = spaceRe.ReplaceAllString(line, " ")
		cleaned = append(cleaned, line)
	}

	return strings.Join(cleaned, "\n")
}

func ParseTaxInvoiceText(raw string) (*tax.TaxInvoice, error) {
	text := normalize(raw)
	lines := strings.Split(text, "\n")

	number := parseReference(text)

	buyerBlock := extractBuyerBlock(lines)
	buyer := parseBuyer(buyerBlock)

	return &tax.TaxInvoice{
		Number: number,
		Buyer:  buyer,
	}, nil
}

func parseBuyer(block []string) *invoice.Party {
	var (
		name    string
		npwp    string
		address []string
	)

	for i := 0; i < len(block); i++ {
		line := block[i]

		if strings.HasPrefix(line, "Nama") {
			name = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			continue
		}

		if strings.HasPrefix(line, "Alamat") {
			addr := strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
			address = append(address, addr)

			// lanjutkan ambil baris alamat berikutnya
			for j := i + 1; j < len(block); j++ {
				if strings.HasPrefix(block[j], "NPWP") {
					break
				}
				address = append(address, block[j])
				i = j
			}
			continue
		}

		if strings.HasPrefix(line, "NPWP") {
			npwp = strings.TrimSpace(strings.SplitN(line, ":", 2)[1])
		}
	}

	addrez := strings.Join(address, " ")

	return &invoice.Party{
		Name:    name,
		Address: &addrez,
		TaxID:   &npwp,
	}
}

func extractBuyerBlock(lines []string) []string {
	var block []string
	in := false

	for _, line := range lines {
		if strings.HasPrefix(line, "Pembeli Barang Kena Pajak") {
			in = true
			continue
		}
		if in {
			if strings.HasPrefix(line, "NPWP") {
				block = append(block, line)
				break
			}
			block = append(block, line)
		}
	}

	return block
}

func parseReference(text string) string {
	m := refRe.FindStringSubmatch(text)
	if len(m) != 2 {
		return ""
	}

	ref := strings.TrimSpace(m[1])

	// buang prefix "Invoice No :" kalau ada
	ref = strings.TrimPrefix(ref, "Invoice No :")
	ref = strings.TrimPrefix(ref, "Invoice No:")
	ref = strings.TrimSpace(ref)

	return ref
}
