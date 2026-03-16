package spreadsheet

import (
	"bytes"
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/sieryo/invoice-extractor/internal/app/document"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
	"github.com/xuri/excelize/v2"
)

type Service struct {
	ingestService *ingest.IngestService
	fileStore     file.FileStore
}

func NewService(ingestService *ingest.IngestService, fileStore file.FileStore) *Service {
	return &Service{
		ingestService: ingestService,
		fileStore:     fileStore,
	}
}

type WorkbookMeta struct {
	DocumentID   string   `json:"documentId"`
	SourceName   string   `json:"sourceName"`
	SheetNames   []string `json:"sheetNames"`
	SheetCount   int      `json:"sheetCount"`
	DefaultSheet string   `json:"defaultSheet,omitempty"`
}

type StreamSheetRowsInput struct {
	SheetName       string `json:"sheetName"`
	HeaderRowNumber int    `json:"headerRowNumber,omitempty"`
	StartRow        int    `json:"startRow,omitempty"`
	MaxRows         int    `json:"maxRows,omitempty"`
}

type SheetRow struct {
	RowNumber int      `json:"rowNumber"`
	Cells     []string `json:"cells"`
}

func (s *Service) GetWorkbookMeta(
	ctx context.Context,
	userID string,
	collectionID string,
	documentID string,
) (*WorkbookMeta, error) {
	workbook, doc, err := s.openWorkbookByDocument(ctx, userID, collectionID, documentID)
	if err != nil {
		return nil, err
	}
	defer workbook.Close()

	sheets := workbook.GetSheetList()
	meta := &WorkbookMeta{
		DocumentID: doc.ID,
		SourceName: doc.SourceName,
		SheetNames: sheets,
		SheetCount: len(sheets),
	}
	if len(sheets) > 0 {
		meta.DefaultSheet = sheets[0]
	}

	return meta, nil
}

func (s *Service) StreamSheetRows(
	ctx context.Context,
	userID string,
	collectionID string,
	documentID string,
	input StreamSheetRowsInput,
	consume func(SheetRow) error,
) error {
	if consume == nil {
		return nil
	}

	sheetName := strings.TrimSpace(input.SheetName)
	if sheetName == "" {
		return ErrSheetNameRequired
	}

	startRow := input.StartRow
	if startRow <= 0 && input.HeaderRowNumber > 0 {
		startRow = input.HeaderRowNumber
	}
	if startRow <= 0 {
		startRow = 1
	}

	maxRows := input.MaxRows
	if maxRows < 0 {
		return ErrInvalidRowRange
	}

	workbook, _, err := s.openWorkbookByDocument(ctx, userID, collectionID, documentID)
	if err != nil {
		return err
	}
	defer workbook.Close()

	if !containsSheet(workbook.GetSheetList(), sheetName) {
		return ErrSheetNotFound
	}

	rows, err := workbook.Rows(sheetName)
	if err != nil {
		return err
	}
	defer rows.Close()

	rowNumber := 0
	emitted := 0
	for rows.Next() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		rowNumber++
		if rowNumber < startRow {
			continue
		}

		cells, err := rows.Columns()
		if err != nil {
			return err
		}

		row := SheetRow{
			RowNumber: rowNumber,
			Cells:     append([]string(nil), cells...),
		}
		if err := consume(row); err != nil {
			return err
		}

		emitted++
		if maxRows > 0 && emitted >= maxRows {
			break
		}
	}

	return nil
}

func (s *Service) openWorkbookByDocument(
	ctx context.Context,
	userID string,
	collectionID string,
	documentID string,
) (*excelize.File, *ingest.DocumentRecord, error) {
	doc, err := s.ingestService.GetDocument(ctx, userID, collectionID, documentID)
	if err != nil {
		return nil, nil, err
	}

	if !isSpreadsheetDocumentType(doc.DocumentType) {
		return nil, nil, ErrUnsupportedDocumentType
	}

	rawRef := ""
	if doc.RawRef != nil {
		rawRef = strings.TrimSpace(*doc.RawRef)
	}
	if rawRef == "" {
		return nil, nil, ErrWorkbookArtifactMissing
	}

	rawName := filepath.Base(rawRef)
	if rawName == "" || rawName == "." {
		return nil, nil, ErrWorkbookArtifactMissing
	}

	data, err := s.fileStore.Read(ctx, collectionID, rawName)
	if err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrWorkbookArtifactMissing, err)
	}

	workbook, err := excelize.OpenReader(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	return workbook, doc, nil
}

func isSpreadsheetDocumentType(docType document.DocumentType) bool {
	return strings.HasPrefix(strings.ToLower(strings.TrimSpace(string(docType))), "xlsx_")
}

func containsSheet(sheetNames []string, target string) bool {
	for _, sheetName := range sheetNames {
		if strings.EqualFold(strings.TrimSpace(sheetName), target) {
			return true
		}
	}
	return false
}
