package pdftool

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"
)

func TestExtractText_GenerateTXTFromAllPDFAssets(t *testing.T) {
	backendRoot := resolveBackendRoot(t)
	repoRoot := filepath.Dir(backendRoot)
	assetsDir := filepath.Join(backendRoot, "assets", "pdf")
	toolDir := filepath.Join(repoRoot, "tools", "pdftotext", "bin")

	if _, err := os.Stat(assetsDir); err != nil {
		t.Fatalf("assets folder not found: %s (%v)", assetsDir, err)
	}
	if _, err := os.Stat(toolDir); err != nil {
		t.Fatalf("pdftotext tool folder not found: %s (%v)", toolDir, err)
	}

	t.Setenv("PATH", toolDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	entries, err := os.ReadDir(assetsDir)
	if err != nil {
		t.Fatalf("failed to read assets dir: %v", err)
	}

	pdfNames := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if strings.EqualFold(filepath.Ext(entry.Name()), ".pdf") {
			pdfNames = append(pdfNames, entry.Name())
		}
	}
	if len(pdfNames) == 0 {
		t.Fatalf("no pdf fixtures found in %s", assetsDir)
	}
	slices.Sort(pdfNames)

	for _, pdfName := range pdfNames {
		pdfPath := filepath.Join(assetsDir, pdfName)
		txtPath := filepath.Join(assetsDir, strings.TrimSuffix(pdfName, filepath.Ext(pdfName))+".txt")

		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		text, extractErr := ExtractText(ctx, pdfPath, DefaultOptions())
		cancel()
		if extractErr != nil {
			t.Fatalf("extract failed for %s: %v", pdfName, extractErr)
		}
		if strings.TrimSpace(text) == "" {
			t.Fatalf("empty extracted text for %s", pdfName)
		}

		if err := os.WriteFile(txtPath, []byte(text), 0o644); err != nil {
			t.Fatalf("failed to write txt for %s: %v", pdfName, err)
		}
	}
}

func resolveBackendRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatalf("failed to resolve current file path")
	}
	// from backend/internal/infra/adapter/pdftool -> backend
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", "..", ".."))
}
