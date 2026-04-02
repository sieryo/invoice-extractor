package parsers

import (
	"context"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/domain/bukpot"
)

type BP21Parser struct{}

func NewBP21Parser() *BP21Parser {
	return &BP21Parser{}
}

func (p *BP21Parser) Kind() bukpot.Kind {
	return bukpot.KindBP21
}

func (p *BP21Parser) Match(text string) bool {
	return strings.Contains(strings.ToUpper(text), "BP21")
}

func (p *BP21Parser) Parse(_ context.Context, text string) (*bukpot.ParsedDocument, error) {
	norm := normalizeText(text)
	lines := strings.Split(norm, "\n")
	top := parseTopFields(lines, "A. IDENTITAS PENERIMA PENGHASILAN")

	kodeObjek, objekPajak := parseObjectFields(
		lines,
		"KODE OBJEK",
		"B.8",
		"B.9",
		"C.",
	)

	referensiJenisRaw := firstLineAfterPrefix(lines, "B.8 Dokumen Referensi Jenis Dokumen :")
	referensiTanggal := parseIndonesianDate(referensiJenisRaw)
	if referensiTanggal == "" {
		referensiTanggal = firstDateInText(referensiJenisRaw)
	}

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
		Kind:        bukpot.KindBP21,
		DocumentTag: deriveDocumentTag(namaPemotong, firstLineAfterPrefix(lines, "A.2 Nama :")),
		BP21: &bukpot.BP21Document{
			NomorBuktiPotong:        top.Nomor,
			MasaPajak:               top.Masa,
			SifatPemotongan:         top.SifatPemotongan,
			StatusBukti:             top.StatusBukti,
			NIKNPWPPenerima:         firstLineAfterPrefix(lines, "A.1 NIK/NPWP :"),
			NamaPenerima:            firstLineAfterPrefix(lines, "A.2 Nama :"),
			NITKUPenerima:           firstLineAfterPrefix(lines, "A.3 NITKU :"),
			JenisFasilitas:          firstLineAfterPrefix(lines, "B.1 Jenis Fasilitas :"),
			KodeObjekPajak:          kodeObjek,
			ObjekPajak:              objekPajak,
			DokumenReferensiJenis:   normalizeDokumenReferensiJenis(referensiJenisRaw),
			DokumenReferensiNomor:   firstLineAfterPrefix(lines, "B.9 Nomor Dokumen :"),
			DokumenReferensiTanggal: referensiTanggal,
			NPWPNIKPemotong:         npwpPemotong,
			NamaPemotong:            namaPemotong,
			TanggalPemotongan:       tanggal,
			NamaPenandatangan:       penandatangan,
		},
	}, nil
}
