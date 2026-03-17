package bukpot

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

type Kind string

const (
	KindBPPU Kind = "bppu"
	KindBP21 Kind = "bp21"
	KindBPA1 Kind = "bpa1"
)

func (k Kind) String() string {
	return string(k)
}

func (k Kind) IsValid() bool {
	switch k {
	case KindBPPU, KindBP21, KindBPA1:
		return true
	default:
		return false
	}
}

func ParseKind(raw string) (Kind, error) {
	k := Kind(strings.ToLower(strings.TrimSpace(raw)))
	if !k.IsValid() {
		return "", fmt.Errorf("invalid bukpot kind: %s", raw)
	}
	return k, nil
}

func (k *Kind) UnmarshalJSON(data []byte) error {
	var raw string
	if err := json.Unmarshal(data, &raw); err != nil {
		return err
	}

	parsed, err := ParseKind(raw)
	if err != nil {
		return err
	}

	*k = parsed
	return nil
}

func (k Kind) MarshalJSON() ([]byte, error) {
	if !k.IsValid() {
		return nil, fmt.Errorf("invalid bukpot kind: %s", k)
	}
	return json.Marshal(k.String())
}

type FileInput struct {
	UploadIndex int
	SourceName  string
	Path        string
}

type ParsedDocument struct {
	Kind        Kind          `json:"kind"`
	DocumentTag string        `json:"documentTag,omitempty"`
	BPPU        *BPPUDocument `json:"bppu,omitempty"`
	BP21        *BP21Document `json:"bp21,omitempty"`
	BPA1        *BPA1Document `json:"bpa1,omitempty"`
}

type BPPUDocument struct {
	NomorBuktiPotong        string `json:"nomorBuktiPotong,omitempty"`
	MasaPajak               string `json:"masaPajak,omitempty"`
	SifatPemotongan         string `json:"sifatPemotongan,omitempty"`
	StatusBukti             string `json:"statusBukti,omitempty"`
	NPWPNIKPenerima         string `json:"npwpNikPenerima,omitempty"`
	NamaPenerima            string `json:"namaPenerima,omitempty"`
	NITKUPenerima           string `json:"nitkuPenerima,omitempty"`
	JenisFasilitas          string `json:"jenisFasilitas,omitempty"`
	JenisPPh                string `json:"jenisPph,omitempty"`
	KodeObjekPajak          string `json:"kodeObjekPajak,omitempty"`
	ObjekPajak              string `json:"objekPajak,omitempty"`
	DokumenReferensiJenis   string `json:"dokumenReferensiJenis,omitempty"`
	DokumenReferensiNomor   string `json:"dokumenReferensiNomor,omitempty"`
	DokumenReferensiTanggal string `json:"dokumenReferensiTanggal,omitempty"`
	NPWPNIKPemotong         string `json:"npwpNikPemotong,omitempty"`
	NamaPemotong            string `json:"namaPemotong,omitempty"`
	TanggalPemotongan       string `json:"tanggalPemotongan,omitempty"`
	NamaPenandatangan       string `json:"namaPenandatangan,omitempty"`
}

type BP21Document struct {
	NomorBuktiPotong        string `json:"nomorBuktiPotong,omitempty"`
	MasaPajak               string `json:"masaPajak,omitempty"`
	SifatPemotongan         string `json:"sifatPemotongan,omitempty"`
	StatusBukti             string `json:"statusBukti,omitempty"`
	NIKNPWPPenerima         string `json:"nikNpwpPenerima,omitempty"`
	NamaPenerima            string `json:"namaPenerima,omitempty"`
	NITKUPenerima           string `json:"nitkuPenerima,omitempty"`
	JenisFasilitas          string `json:"jenisFasilitas,omitempty"`
	KodeObjekPajak          string `json:"kodeObjekPajak,omitempty"`
	ObjekPajak              string `json:"objekPajak,omitempty"`
	DokumenReferensiJenis   string `json:"dokumenReferensiJenis,omitempty"`
	DokumenReferensiNomor   string `json:"dokumenReferensiNomor,omitempty"`
	DokumenReferensiTanggal string `json:"dokumenReferensiTanggal,omitempty"`
	NPWPNIKPemotong         string `json:"npwpNikPemotong,omitempty"`
	NamaPemotong            string `json:"namaPemotong,omitempty"`
	TanggalPemotongan       string `json:"tanggalPemotongan,omitempty"`
	NamaPenandatangan       string `json:"namaPenandatangan,omitempty"`
}

type BPA1Document struct {
	NomorBuktiPotong   string `json:"nomorBuktiPotong,omitempty"`
	PeriodePenghasilan string `json:"periodePenghasilan,omitempty"`
	SifatPemotongan    string `json:"sifatPemotongan,omitempty"`
	StatusBukti        string `json:"statusBukti,omitempty"`
	NIKNPWPPenerima    string `json:"nikNpwpPenerima,omitempty"`
	NamaPenerima       string `json:"namaPenerima,omitempty"`
	Posisi             string `json:"posisi,omitempty"`
	StatusPTKP         string `json:"statusPtkp,omitempty"`
	NPWPNIKPemotong    string `json:"npwpNikPemotong,omitempty"`
	NamaPemotong       string `json:"namaPemotong,omitempty"`
	TanggalPemotongan  string `json:"tanggalPemotongan,omitempty"`
	NamaPenandatangan  string `json:"namaPenandatangan,omitempty"`
}

type ParsedFile struct {
	Input FileInput       `json:"input"`
	Data  *ParsedDocument `json:"data,omitempty"`
	Error *string         `json:"error,omitempty"`
}

type Parser interface {
	Kind() Kind
	Match(text string) bool
	Parse(ctx context.Context, text string) (*ParsedDocument, error)
}
