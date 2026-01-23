package pdftool

import (
	"os"
	"os/exec"
	"path/filepath"
)

func ResolvePDFToTextPath() (string, error) {
	exePath, err := os.Executable()
	if err != nil {
		return "", err
	}

	appDir := filepath.Dir(exePath)

	// 1. Production: bundled binary
	bundled := filepath.Join(appDir, "bin", "pdftotext.exe")
	if _, err := os.Stat(bundled); err == nil {
		return bundled, nil
	}

	// 2. Development: system PATH
	if path, err := exec.LookPath("pdftotext"); err == nil {
		return path, nil
	}

	return "", ErrPDFToTextNotFound
}
