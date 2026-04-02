package parsers

import (
	"context"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/domain/bukpot"
)

type BPPUParser struct{}

func NewBPPUParser() *BPPUParser {
	return &BPPUParser{}
}

func (p *BPPUParser) Kind() bukpot.Kind {
	return bukpot.KindBPPU
}

func (p *BPPUParser) Match(text string) bool {
	return strings.Contains(strings.ToUpper(text), "BPPU")
}

func (p *BPPUParser) Parse(_ context.Context, text string) (*bukpot.ParsedDocument, error) {
	norm := normalizeText(text)
	lines := strings.Split(norm, "\n")
	top := parseTopFields(lines, "A. IDENTITAS WAJIB PAJAK")

	kodeObjek, objekPajak := parseObjectFields(
		lines,
		"KODE OBJEK PAJAK",
		"B.8",
		"B.9",
		"B.10",
		"B.11",
		"C.",
	)

	npwpPemotong, namaPemotong := extractNPWPNIKAndName(
		firstLineAfterPrefix(lines, "C.2 NOMOR IDENTITAS TEMPAT KEGIATAN :"),
	)
	if npwpPemotong == "" || namaPemotong == "" {
		c1NPWP, c1Name := extractNPWPNIKAndName(firstLineAfterPrefix(lines, "C.1 NPWP / NIK :"))
		if npwpPemotong == "" {
			npwpPemotong = c1NPWP
		}
		if namaPemotong == "" {
			namaPemotong = c1Name
		}
	}

	tanggal := parseIndonesianDate(firstLineAfterPrefix(lines, "C.4 TANGGAL :"))
	if tanggal == "" {
		tanggal = firstDateInText(strings.Join(lines, "\n"))
	}

	referensiJenisRaw := firstLineAfterPrefix(lines, "B.8 Dokumen Dasar Bukti Jenis Dokumen :")
	referensiTanggal := parseIndonesianDate(referensiJenisRaw)
	if referensiTanggal == "" {
		referensiTanggal = firstDateInText(referensiJenisRaw)
	}

	return &bukpot.ParsedDocument{
		Kind:        bukpot.KindBPPU,
		DocumentTag: deriveDocumentTag(namaPemotong, firstLineAfterPrefix(lines, "A.2 NAMA :")),
		BPPU: &bukpot.BPPUDocument{
			NomorBuktiPotong:        top.Nomor,
			MasaPajak:               top.Masa,
			SifatPemotongan:         top.SifatPemotongan,
			StatusBukti:             top.StatusBukti,
			NPWPNIKPenerima:         firstLineAfterPrefix(lines, "A.1 NPWP / NIK :"),
			NamaPenerima:            firstLineAfterPrefix(lines, "A.2 NAMA :"),
			NITKUPenerima:           firstLineAfterPrefix(lines, "A.3 NOMOR IDENTITAS :"),
			JenisFasilitas:          firstLineAfterPrefix(lines, "B.1 Jenis Fasilitas :"),
			JenisPPh:                firstLineAfterPrefix(lines, "B.2 Jenis PPh :"),
			KodeObjekPajak:          kodeObjek,
			ObjekPajak:              objekPajak,
			DokumenReferensiJenis:   normalizeDokumenReferensiJenis(referensiJenisRaw),
			DokumenReferensiNomor:   firstLineAfterPrefix(lines, "B.9 Nomor Dokumen :"),
			DokumenReferensiTanggal: referensiTanggal,
			NPWPNIKPemotong:         npwpPemotong,
			NamaPemotong:            namaPemotong,
			TanggalPemotongan:       tanggal,
			NamaPenandatangan:       firstLineAfterPrefix(lines, "C.5 NAMA PENANDATANGAN :"),
		},
	}, nil
}
