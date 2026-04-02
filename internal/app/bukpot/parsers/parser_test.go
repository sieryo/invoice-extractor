package parsers

import (
	"context"
	"encoding/binary"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"unicode/utf16"

	"github.com/sieryo/invoice-extractor/internal/domain/bukpot"
)

func TestBP21Parser_ParseFromRawTXT(t *testing.T) {
	text := mustReadFixture(t, "example_bp21.txt")
	parser := NewBP21Parser()

	if !parser.Match(text) {
		t.Fatalf("expected BP21 parser to match fixture")
	}

	out, err := parser.Parse(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if out.Kind != bukpot.KindBP21 {
		t.Fatalf("unexpected kind: %s", out.Kind)
	}
	if out.BP21 == nil {
		t.Fatalf("expected bp21 payload")
	}
	if strings.TrimSpace(out.DocumentTag) == "" {
		t.Fatalf("expected document tag to be populated")
	}

	got := out.BP21
	if got.NomorBuktiPotong != "25078TO5M" {
		t.Fatalf("unexpected nomor bukti: %q", got.NomorBuktiPotong)
	}
	if got.MasaPajak != "11-2025" {
		t.Fatalf("unexpected masa pajak: %q", got.MasaPajak)
	}
	if got.StatusBukti != "NORMAL" {
		t.Fatalf("unexpected status bukti: %q", got.StatusBukti)
	}
	if got.NamaPenerima != "SYIVA HERLIANA" {
		t.Fatalf("unexpected nama penerima: %q", got.NamaPenerima)
	}
	if got.KodeObjekPajak != "21-100-20" {
		t.Fatalf("unexpected kode objek pajak: %q", got.KodeObjekPajak)
	}
	if !strings.Contains(strings.ToLower(got.ObjekPajak), "imbalan kepada pemberi jasa") {
		t.Fatalf("unexpected objek pajak: %q", got.ObjekPajak)
	}
}

func TestBPA1Parser_ParseFromRawTXT(t *testing.T) {
	text := mustReadFixture(t, "example_bpa1.txt")
	parser := NewBPA1Parser()

	if !parser.Match(text) {
		t.Fatalf("expected BPA1 parser to match fixture")
	}

	out, err := parser.Parse(context.Background(), text)
	if err != nil {
		t.Fatalf("unexpected parse error: %v", err)
	}
	if out.Kind != bukpot.KindBPA1 {
		t.Fatalf("unexpected kind: %s", out.Kind)
	}
	if out.BPA1 == nil {
		t.Fatalf("expected bpa1 payload")
	}
	if strings.TrimSpace(out.DocumentTag) == "" {
		t.Fatalf("expected document tag to be populated")
	}

	got := out.BPA1
	if got.NomorBuktiPotong != "2507TJSDP" {
		t.Fatalf("unexpected nomor bukti: %q", got.NomorBuktiPotong)
	}
	if got.PeriodePenghasilan != "01-2025-12-2025" {
		t.Fatalf("unexpected periode penghasilan: %q", got.PeriodePenghasilan)
	}
	if got.StatusBukti != "NORMAL" {
		t.Fatalf("unexpected status bukti: %q", got.StatusBukti)
	}
	if got.NamaPenerima != "Aprilia Nuraeni" {
		t.Fatalf("unexpected nama penerima: %q", got.NamaPenerima)
	}
	if got.Posisi != "STAFF" {
		t.Fatalf("unexpected posisi: %q", got.Posisi)
	}
	if got.StatusPTKP != "TK0" {
		t.Fatalf("unexpected status ptkp: %q", got.StatusPTKP)
	}
}

func TestBPPUParser_ParseFromRawTXT(t *testing.T) {
	parser := NewBPPUParser()
	cases := []struct {
		fixture       string
		nomor         string
		masa          string
		sifat         string
		status        string
		namaPenerima  string
		kodeObjek     string
		objekContains string
	}{
		{
			fixture:       "example_bppu.txt",
			nomor:         "2507AAPI0",
			masa:          "11-2025",
			sifat:         "TIDAK FINAL",
			status:        "NORMAL",
			namaPenerima:  "HIBURAN JALAN KELUAR",
			kodeObjek:     "24-104-23",
			objekContains: "jasa pembuatan sarana promosi",
		},
		{
			fixture:       "example_bppu2.txt",
			nomor:         "2506XITO1",
			masa:          "11-2025",
			sifat:         "TIDAK FINAL",
			status:        "DIBATALKAN",
			namaPenerima:  "FAJAR MITRA INDAH",
			kodeObjek:     "24-100-01",
			objekContains: "hadiah, penghargaan, bonus",
		},
		{
			fixture:       "example_bppu3.txt",
			nomor:         "2504PLXAE",
			masa:          "08-2025",
			sifat:         "TIDAK FINAL",
			status:        "PEMBETULAN",
			namaPenerima:  "DUA CAHAYA KREASI",
			kodeObjek:     "24-104-23",
			objekContains: "jasa pembuatan sarana promosi",
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			text := mustReadFixture(t, tc.fixture)

			if !parser.Match(text) {
				t.Fatalf("expected BPPU parser to match fixture")
			}

			out, err := parser.Parse(context.Background(), text)
			if err != nil {
				t.Fatalf("unexpected parse error: %v", err)
			}
			if out.Kind != bukpot.KindBPPU {
				t.Fatalf("unexpected kind: %s", out.Kind)
			}
			if out.BPPU == nil {
				t.Fatalf("expected bppu payload")
			}
			if strings.TrimSpace(out.DocumentTag) == "" {
				t.Fatalf("expected document tag to be populated")
			}

			got := out.BPPU
			if got.NomorBuktiPotong != tc.nomor {
				t.Fatalf("unexpected nomor bukti: %q", got.NomorBuktiPotong)
			}
			if got.MasaPajak != tc.masa {
				t.Fatalf("unexpected masa pajak: %q", got.MasaPajak)
			}
			if got.SifatPemotongan != tc.sifat {
				t.Fatalf("unexpected sifat pemotongan: %q", got.SifatPemotongan)
			}
			if got.StatusBukti != tc.status {
				t.Fatalf("unexpected status bukti: %q", got.StatusBukti)
			}
			if got.NamaPenerima != tc.namaPenerima {
				t.Fatalf("unexpected nama penerima: %q", got.NamaPenerima)
			}
			if got.KodeObjekPajak != tc.kodeObjek {
				t.Fatalf("unexpected kode objek pajak: %q", got.KodeObjekPajak)
			}
			if !strings.Contains(strings.ToLower(got.ObjekPajak), tc.objekContains) {
				t.Fatalf("unexpected objek pajak: %q", got.ObjekPajak)
			}
		})
	}
}

func TestGenerateParsedJSONFromFixtures(t *testing.T) {
	cases := []struct {
		fixture string
		parse   func(string) (*bukpot.ParsedDocument, error)
	}{
		{
			fixture: "example_bp21.txt",
			parse: func(text string) (*bukpot.ParsedDocument, error) {
				return NewBP21Parser().Parse(context.Background(), text)
			},
		},
		{
			fixture: "example_bpa1.txt",
			parse: func(text string) (*bukpot.ParsedDocument, error) {
				return NewBPA1Parser().Parse(context.Background(), text)
			},
		},
		{
			fixture: "example_bppu.txt",
			parse: func(text string) (*bukpot.ParsedDocument, error) {
				return NewBPPUParser().Parse(context.Background(), text)
			},
		},
		{
			fixture: "example_bppu2.txt",
			parse: func(text string) (*bukpot.ParsedDocument, error) {
				return NewBPPUParser().Parse(context.Background(), text)
			},
		},
		{
			fixture: "example_bppu3.txt",
			parse: func(text string) (*bukpot.ParsedDocument, error) {
				return NewBPPUParser().Parse(context.Background(), text)
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.fixture, func(t *testing.T) {
			path := fixturePath(t, tc.fixture)
			raw, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read fixture %s: %v", tc.fixture, err)
			}

			doc, err := tc.parse(decodeFixtureText(raw))
			if err != nil {
				t.Fatalf("failed to parse fixture %s: %v", tc.fixture, err)
			}
			if doc == nil {
				t.Fatalf("nil parsed document for fixture %s", tc.fixture)
			}

			payload, err := json.MarshalIndent(doc, "", "  ")
			if err != nil {
				t.Fatalf("failed to marshal parsed json for fixture %s: %v", tc.fixture, err)
			}

			outPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".parsed.json"
			t.Logf("writing parsed json: %s", outPath)
			if err := os.WriteFile(outPath, payload, 0o644); err != nil {
				t.Fatalf("failed to write parsed json %s: %v", outPath, err)
			}
		})
	}
}

func mustReadFixture(t *testing.T, name string) string {
	t.Helper()
	path := fixturePath(t, name)
	bytes, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read fixture %s: %v", name, err)
	}
	return decodeFixtureText(bytes)
}

func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("unable to resolve caller path")
	}
	return filepath.Join(filepath.Dir(file), "..", "..", "..", "..", "assets", "pdf", name)
}

func decodeFixtureText(raw []byte) string {
	if len(raw) >= 2 {
		if raw[0] == 0xFF && raw[1] == 0xFE {
			return decodeUTF16(raw[2:], binary.LittleEndian)
		}
		if raw[0] == 0xFE && raw[1] == 0xFF {
			return decodeUTF16(raw[2:], binary.BigEndian)
		}
	}

	zeroCount := 0
	for _, b := range raw {
		if b == 0x00 {
			zeroCount++
		}
	}
	if zeroCount > len(raw)/4 {
		return decodeUTF16(raw, binary.LittleEndian)
	}

	return string(raw)
}

func decodeUTF16(raw []byte, order binary.ByteOrder) string {
	if len(raw)%2 != 0 {
		raw = raw[:len(raw)-1]
	}
	if len(raw) == 0 {
		return ""
	}

	u16 := make([]uint16, 0, len(raw)/2)
	for i := 0; i < len(raw); i += 2 {
		u16 = append(u16, order.Uint16(raw[i:i+2]))
	}
	return string(utf16.Decode(u16))
}
