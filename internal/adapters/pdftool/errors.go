package pdftool

import "errors"

var (
	ErrPDFToTextNotFound = errors.New("pdftotext not found")
	ErrExtractFailed     = errors.New("pdftotext execution failed")
)
