package spreadsheet

import "errors"

var (
	ErrUnsupportedDocumentType = errors.New("unsupported document type for spreadsheet workbook")
	ErrWorkbookArtifactMissing = errors.New("spreadsheet workbook artifact is missing")
	ErrSheetNameRequired       = errors.New("sheetName is required")
	ErrSheetNotFound           = errors.New("sheet not found in workbook")
	ErrInvalidRowRange         = errors.New("invalid row range")
)
