package parsers

import (
	"context"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/domain/bukpot"
)

type BPA1Parser struct{}

func NewBPA1Parser() *BPA1Parser {
	return &BPA1Parser{}
}

func (p *BPA1Parser) Kind() bukpot.Kind {
	return bukpot.KindBPA1
}

func (p *BPA1Parser) Match(text string) bool {
	return strings.Contains(strings.ToUpper(text), "BPA1")
}

func (p *BPA1Parser) Parse(_ context.Context, text string) (*bukpot.ParsedDocument, error) {
	norm := normalizeText(text)
	lines := strings.Split(norm, "\n")
	top := parseTopFields(lines, "A. IDENTITAS PENERIMA PENGHASILAN")

	npwpPemotong, namaPemotong := extractNPWPNIKAndName(firstLineAfterPrefix(lines, "C.1 NPWP/NIK :"))
	if namaPemotong == "" {
		_, fallbackName := extractNPWPNIKAndName(firstLineAfterPrefix(lines, "C.2 NITKU atau Nomor Identitas Subunit Organisasi :"))
		namaPemotong = fallbackName
	}

	tanggal := parseIndonesianDate(firstLineAfterPrefix(lines, "C.4 Tanggal :"))
	if tanggal == "" {
		tanggal = parseIndonesianDate(firstLineAfterPrefix(lines, "C.3 Nama Pemotong :"))
	}

	penandatangan := firstLineAfterPrefix(lines, "C.5 Nama Penandatangan :")
	if penandatangan == "" || strings.HasPrefix(strings.ToLower(penandatangan), "dengan ini") {
		penandatangan = firstLineAfterPrefix(lines, "C.4 Tanggal :")
		if parseIndonesianDate(penandatangan) != "" {
			penandatangan = ""
		}
	}

	return &bukpot.ParsedDocument{
		Kind:        bukpot.KindBPA1,
		DocumentTag: deriveDocumentTag(namaPemotong, firstLineAfterPrefix(lines, "A.2 Nama :")),
		BPA1: &bukpot.BPA1Document{
			NomorBuktiPotong:   top.Nomor,
			PeriodePenghasilan: top.Masa,
			SifatPemotongan:    top.SifatPemotongan,
			StatusBukti:        top.StatusBukti,
			NIKNPWPPenerima:    firstLineAfterPrefix(lines, "A.1 NIK/NPWP :"),
			NamaPenerima:       firstLineAfterPrefix(lines, "A.2 Nama :"),
			Posisi:             extractPosisi(lines),
			StatusPTKP:         extractStatusPTKP(lines),
			NPWPNIKPemotong:    npwpPemotong,
			NamaPemotong:       namaPemotong,
			TanggalPemotongan:  tanggal,
			NamaPenandatangan:  penandatangan,
		},
	}, nil
}

func extractPosisi(lines []string) string {
	val := firstLineAfterPrefix(lines, "A.6 Posisi :")
	if val == "" {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(val, " A.9", 2)[0])
}

func extractStatusPTKP(lines []string) string {
	val := firstLineAfterPrefix(lines, "A.5 Status PTKP :")
	if val == "" {
		return ""
	}
	return strings.TrimSpace(strings.SplitN(val, " A.8", 2)[0])
}
