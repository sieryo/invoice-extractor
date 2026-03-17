package extract

import (
	"errors"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/invoice/tax"
)

var errInvoiceParse = errors.New("unable to parse tax invoice")

var invoiceNumberPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?im)kode\s+dan\s+nomor\s+seri\s+faktur\s+pajak\s*[:\-]?\s*([0-9]{13,})`),
	regexp.MustCompile(`(?im)(?:nomor\s+faktur|invoice\s*number|faktur)\s*[:\-]?\s*([0-9.\-]{8,})`),
	regexp.MustCompile(`\b([0-9]{3}\.[0-9]{3}-[0-9]{2}\.[0-9]{8})\b`),
}

var monthMap = map[string]time.Month{
	"januari":   time.January,
	"februari":  time.February,
	"maret":     time.March,
	"april":     time.April,
	"mei":       time.May,
	"juni":      time.June,
	"juli":      time.July,
	"agustus":   time.August,
	"september": time.September,
	"oktober":   time.October,
	"november":  time.November,
	"desember":  time.December,
}

var dateIndonesianRegex = regexp.MustCompile(`(?i)\b([0-3]?[0-9])\s+(januari|februari|maret|april|mei|juni|juli|agustus|september|oktober|november|desember)\s+([12][0-9]{3})\b`)
var numericDateRegexes = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(?:tanggal|date)\s*[:\-]?\s*([0-3]?[0-9][/\-.][0-1]?[0-9][/\-.][0-9]{2,4})`),
	regexp.MustCompile(`\b([0-3]?[0-9][/\-.][0-1]?[0-9][/\-.][0-9]{2,4})\b`),
}
var npwpRegex = regexp.MustCompile(`(?im)\bnpwp\s*:\s*([0-9.\-]{15,30}|[0-9]{15,20})`)
var nameRegex = regexp.MustCompile(`(?im)\bnama\s*:\s*(.+)$`)
var referenceRegex = regexp.MustCompile(`(?im)\(\s*referensi\s*:\s*([^)]+)?\)`)
var multiSpaceRegex = regexp.MustCompile(`\s+`)
var itemLineRegex = regexp.MustCompile(`(?i)^([0-9]+)\s+([0-9A-Za-z]+)\s+rp\s*([0-9.,]+)\s*x\s*([0-9.,]+)\s+([A-Za-z]+)\s+([0-9.,]+)$`)

func parseCoretaxFromText(filename string, rawText string) (tax.TaxInvoice, []string, error) {
	cleaned := normalizeTaxInvoiceText(rawText)
	anomalies := []string{}

	number := firstMatch(cleaned, invoiceNumberPatterns)
	if number == "" {
		anomalies = append(anomalies, "invoice number not detected")
	}

	invoiceDate, foundDate := findDate(cleaned)
	if !foundDate {
		anomalies = append(anomalies, "invoice date not detected")
	}

	sellerSection, _ := sectionBetween(cleaned, "pengusaha kena pajak:", "pembeli barang kena pajak/penerima jasa kena pajak:")
	buyerSection, _ := sectionBetweenAnyEnd(cleaned, "pembeli barang kena pajak/penerima jasa kena pajak:", []string{"\nno.", "\nno ", "\nkode", "\nharga jual"})

	sellerName := extractNameFromSection(sellerSection)
	buyerName := extractNameFromSection(buyerSection)
	sellerNPWP := extractNPWPFromSection(sellerSection)
	buyerNPWP := extractNPWPFromSection(buyerSection)

	if sellerName == "" {
		anomalies = append(anomalies, "seller name not detected")
	}
	if buyerName == "" {
		anomalies = append(anomalies, "buyer name not detected")
	}
	if sellerNPWP == "" {
		anomalies = append(anomalies, "seller NPWP not detected")
	}
	if buyerNPWP == "" {
		anomalies = append(anomalies, "buyer NPWP not detected")
	}

	dpp := findAmount(cleaned, []string{"dasar pengenaan pajak", "dpp"})
	ppn := findAmount(cleaned, []string{"jumlah ppn", "pajak pertambahan nilai", "ppn"})
	total := findAmount(cleaned, []string{"harga jual / penggantian / uang muka / termin", "jumlah yang harus dibayar", "total"})
	downPaymentReceived := findAmountInlineOnly(cleaned, "dikurangi uang muka yang telah diterima")
	if total == 0 && dpp > 0 {
		total = dpp + ppn
		anomalies = append(anomalies, "total inferred from dpp + ppn")
	}
	if dpp == 0 {
		anomalies = append(anomalies, "DPP not detected")
	}
	if ppn == 0 {
		anomalies = append(anomalies, "PPN not detected")
	}
	if total == 0 {
		anomalies = append(anomalies, "total amount not detected")
	}

	items := extractItems(cleaned)
	if len(items) == 0 {
		anomalies = append(anomalies, "invoice items not detected")
	}
	reference := extractReference(cleaned)
	// Sementara: DPP method + DPP verification dinonaktifkan dulu.
	// Field tetap diisi agar payload stabil, tapi tidak dipakai untuk warning.
	dppMethod := tax.DPPMethodUnknown
	dppVerification := tax.DPPVerification{
		Status:     tax.DPPVerificationOK,
		Threshold:  0,
		BaseAmount: 0,
		ExpectedA:  0,
		ExpectedB:  0,
		DiffA:      0,
		DiffB:      0,
		Message:    "temporary: dpp verification is disabled",
	}

	legacyNumber := strings.TrimSpace(reference)
	if legacyNumber == "" {
		legacyNumber = strings.TrimSpace(number)
	}

	var legacyBuyer *invoice.Party
	if strings.TrimSpace(buyerName) != "" || strings.TrimSpace(buyerNPWP) != "" {
		legacyBuyer = &invoice.Party{
			Name: strings.TrimSpace(buyerName),
		}
		if strings.TrimSpace(buyerNPWP) != "" {
			npwp := strings.TrimSpace(buyerNPWP)
			legacyBuyer.TaxID = &npwp
		}
	}

	invoiceData := tax.TaxInvoice{
		SourceFile:          filename,
		InvoiceNumber:       number,
		InvoiceDate:         invoiceDate,
		SellerName:          sellerName,
		SellerNPWP:          sellerNPWP,
		BuyerName:           buyerName,
		BuyerNPWP:           buyerNPWP,
		References:          reference,
		DownPaymentReceived: downPaymentReceived,
		DPPMethod:           dppMethod,
		DPPVerification:     dppVerification,
		DPP:                 dpp,
		PPN:                 ppn,
		Total:               total,
		Currency:            "IDR",
		Items:               items,
		Number:              legacyNumber,
		Buyer:               legacyBuyer,
	}
	invoiceData.IncludeInExport = invoiceData.InvoiceNumber != "" && invoiceData.BuyerName != ""
	if !invoiceData.IncludeInExport {
		invoiceData.ExclusionReason = "missing mandatory invoice fields for export"
	}

	if invoiceData.InvoiceNumber == "" {
		return invoiceData, anomalies, fmt.Errorf("%w: mandatory field invoice number missing", errInvoiceParse)
	}
	return invoiceData, anomalies, nil
}

func normalizeTaxInvoiceText(s string) string {
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")
	lines := strings.Split(s, "\n")
	normalizedLines := make([]string, 0, len(lines))
	previousEmpty := false
	for _, line := range lines {
		line = strings.ReplaceAll(line, "\u00a0", " ")
		line = strings.TrimSpace(line)
		if line == "" {
			if !previousEmpty {
				normalizedLines = append(normalizedLines, "")
			}
			previousEmpty = true
			continue
		}
		normalizedLines = append(normalizedLines, multiSpaceRegex.ReplaceAllString(line, " "))
		previousEmpty = false
	}
	return strings.TrimSpace(strings.Join(normalizedLines, "\n"))
}

func firstMatch(text string, patterns []*regexp.Regexp) string {
	for _, pattern := range patterns {
		match := pattern.FindStringSubmatch(text)
		if len(match) > 1 {
			return strings.TrimSpace(match[1])
		}
	}
	return ""
}

func findDate(text string) (time.Time, bool) {
	anchor := "secara elektronik sehingga tidak diperlukan tanda tangan basah pada faktur pajak ini."
	lower := strings.ToLower(text)
	if idx := strings.Index(lower, anchor); idx >= 0 {
		window := text[idx:]
		if len(window) > 400 {
			window = window[:400]
		}
		if parsed, ok := findIndonesianDate(window); ok {
			return parsed, true
		}
	}

	if parsed, ok := findIndonesianDate(text); ok {
		return parsed, true
	}

	layouts := []string{
		"2-1-2006", "02-01-2006", "2/1/2006", "02/01/2006",
		"2.1.2006", "02.01.2006", "2-1-06", "02-01-06",
		"2/1/06", "02/01/06", "2.1.06", "02.01.06",
	}
	for _, pattern := range numericDateRegexes {
		match := pattern.FindStringSubmatch(text)
		if len(match) < 2 {
			continue
		}
		dateValue := strings.ReplaceAll(match[1], "/", "-")
		dateValue = strings.ReplaceAll(dateValue, ".", "-")
		for _, layout := range layouts {
			parsed, err := time.Parse(layout, dateValue)
			if err == nil {
				return parsed, true
			}
		}
	}

	return time.Time{}, false
}

func findIndonesianDate(text string) (time.Time, bool) {
	match := dateIndonesianRegex.FindStringSubmatch(text)
	if len(match) != 4 {
		return time.Time{}, false
	}

	day, dayErr := strconv.Atoi(match[1])
	year, yearErr := strconv.Atoi(match[3])
	month, monthOK := monthMap[strings.ToLower(match[2])]
	if dayErr != nil || yearErr != nil || !monthOK {
		return time.Time{}, false
	}
	return time.Date(year, month, day, 0, 0, 0, 0, time.Local), true
}

func sectionBetween(text, startMarker, endMarker string) (string, bool) {
	return sectionBetweenAnyEnd(text, startMarker, []string{endMarker})
}

func sectionBetweenAnyEnd(text, startMarker string, endMarkers []string) (string, bool) {
	lowerText := strings.ToLower(text)
	startIdx := strings.Index(lowerText, strings.ToLower(startMarker))
	if startIdx < 0 {
		return "", false
	}

	sectionStart := startIdx + len(startMarker)
	sectionEnd := len(text)
	for _, endMarker := range endMarkers {
		if endMarker == "" {
			continue
		}
		if idx := strings.Index(lowerText[sectionStart:], strings.ToLower(endMarker)); idx >= 0 {
			candidate := sectionStart + idx
			if candidate < sectionEnd {
				sectionEnd = candidate
			}
		}
	}

	return strings.TrimSpace(text[sectionStart:sectionEnd]), true
}

func extractNameFromSection(section string) string {
	match := nameRegex.FindStringSubmatch(section)
	if len(match) < 2 {
		return ""
	}
	name := strings.TrimSpace(match[1])
	if hash := strings.Index(name, "#"); hash >= 0 {
		name = strings.TrimSpace(name[:hash])
	}
	return name
}

func extractNPWPFromSection(section string) string {
	match := npwpRegex.FindStringSubmatch(section)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func findAmount(text string, keywords []string) float64 {
	lines := strings.Split(text, "\n")
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		lower := strings.ToLower(line)
		for _, keyword := range keywords {
			if strings.Contains(lower, keyword) {
				value := parseLastNumber(line)
				if value > 0 {
					return value
				}
				if i+1 < len(lines) {
					value = parseLastNumber(lines[i+1])
					if value > 0 {
						return value
					}
				}
			}
		}
	}
	return 0
}

func findAmountInlineOnly(text, keyword string) float64 {
	lines := strings.Split(text, "\n")
	needle := strings.ToLower(strings.TrimSpace(keyword))
	for _, rawLine := range lines {
		line := strings.TrimSpace(rawLine)
		if line == "" {
			continue
		}
		lower := strings.ToLower(line)
		if !strings.Contains(lower, needle) {
			continue
		}
		return parseLastNumber(line)
	}
	return 0
}

func parseLastNumber(line string) float64 {
	re := regexp.MustCompile(`([0-9][0-9.,]*)`)
	matches := re.FindAllStringSubmatch(line, -1)
	if len(matches) == 0 {
		return 0
	}
	return parseLocaleNumber(matches[len(matches)-1][1])
}

func parseLocaleNumber(value string) float64 {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, ".", "")
	value = strings.ReplaceAll(value, ",", ".")
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return parsed
}

func extractItems(text string) []tax.TaxInvoiceItem {
	lines := strings.Split(text, "\n")
	items := make([]tax.TaxInvoiceItem, 0)
	for i := range lines {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		match := itemLineRegex.FindStringSubmatch(strings.ToLower(line))
		if len(match) != 7 {
			continue
		}

		lineNo, _ := strconv.Atoi(match[1])
		unitPrice := parseLocaleNumber(match[3])
		qty := parseLocaleNumber(match[4])
		lineTotal := parseLocaleNumber(match[6])
		description := collectItemDescription(lines, i)

		items = append(items, tax.TaxInvoiceItem{
			LineNo:      lineNo,
			ItemCode:    strings.TrimSpace(match[2]),
			Description: description,
			Quantity:    qty,
			Unit:        strings.TrimSpace(match[5]),
			UnitPrice:   unitPrice,
			LineTotal:   lineTotal,
		})
	}
	return items
}

func collectItemDescription(lines []string, itemLineIndex int) string {
	collected := make([]string, 0, 3)
	for i := itemLineIndex - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}

		lower := strings.ToLower(line)
		if isItemDescriptionNoise(lower) {
			if len(collected) > 0 {
				break
			}
			continue
		}

		collected = append([]string{line}, collected...)
		if len(collected) >= 3 {
			break
		}
	}

	if len(collected) == 0 {
		return "-"
	}

	return strings.TrimSpace(strings.Join(collected, " "))
}

func extractReference(text string) string {
	match := referenceRegex.FindStringSubmatch(text)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func isItemDescriptionNoise(lowerLine string) bool {
	if itemLineRegex.MatchString(lowerLine) {
		return true
	}
	identityPrefixes := []string{
		"npwp",
		"nama :",
		"alamat",
		"email",
		"nik",
		"nomor paspor",
		"identitas lain",
		"pengusaha kena pajak",
		"pembeli barang kena pajak",
	}
	for _, prefix := range identityPrefixes {
		if strings.HasPrefix(lowerLine, prefix) {
			return true
		}
	}

	noiseTerms := []string{
		"potongan harga",
		"ppnbm",
		"nama barang kena pajak",
		"harga jual / penggantian / uang muka / termin",
		"barang/",
		"(rp)",
	}
	for _, term := range noiseTerms {
		if strings.Contains(lowerLine, term) {
			return true
		}
	}
	if strings.HasPrefix(lowerLine, "no.") || strings.HasPrefix(lowerLine, "no ") || strings.HasPrefix(lowerLine, "kode") {
		return true
	}
	return false
}

func detectCoretaxDPPMethod(dpp, total, downPaymentReceived float64, items []tax.TaxInvoiceItem) (tax.DPPMethod, tax.DPPVerification) {
	itemCount := len(items)
	if itemCount < 1 {
		itemCount = 1
	}

	baseAmount := total
	if baseAmount <= 0 {
		baseAmount = sumCoretaxItemTotals(items)
	}
	if baseAmount < 0 {
		baseAmount = 0
	}

	threshold := float64(itemCount * 2)
	expectedA := (baseAmount * (11.0 / 12.0)) - downPaymentReceived
	expectedB := (baseAmount - downPaymentReceived) * (11.0 / 12.0)
	if expectedA < 0 {
		expectedA = 0
	}
	if expectedB < 0 {
		expectedB = 0
	}

	diffA := absFloat(expectedA - dpp)
	diffB := absFloat(expectedB - dpp)

	verification := tax.DPPVerification{
		Status:     tax.DPPVerificationError,
		Threshold:  threshold,
		BaseAmount: baseAmount,
		ExpectedA:  expectedA,
		ExpectedB:  expectedB,
		DiffA:      diffA,
		DiffB:      diffB,
	}

	if downPaymentReceived <= 0 {
		if diffB <= threshold {
			verification.Status = tax.DPPVerificationOK
			verification.Message = "DP=0, treated as method B and matched"
			return tax.DPPMethodB, verification
		}
		verification.Message = "DP=0 but DPP does not match expected method B"
		return tax.DPPMethodUnknown, verification
	}

	if diffB <= threshold {
		verification.Status = tax.DPPVerificationOK
		verification.Message = "DP method B (baku) matched"
		return tax.DPPMethodB, verification
	}
	if diffA <= threshold {
		verification.Status = tax.DPPVerificationWarning
		verification.Message = "DP method A (non-baku) matched"
		return tax.DPPMethodA, verification
	}

	verification.Message = "DPP cannot be explained by method A or B within threshold"
	return tax.DPPMethodUnknown, verification
}

func absFloat(value float64) float64 {
	return math.Abs(value)
}

func sumCoretaxItemTotals(items []tax.TaxInvoiceItem) float64 {
	if len(items) == 0 {
		return 0
	}
	total := 0.0
	for _, item := range items {
		total += item.LineTotal
	}
	return total
}
