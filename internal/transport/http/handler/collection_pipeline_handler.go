package handler

import (
	"bufio"
	"encoding/json"
	"errors"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	"github.com/sieryo/invoice-extractor/internal/app/spreadsheet"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type CollectionPipelineHandler struct {
	ingestService  *ingest.IngestService
	spreadsheet    *spreadsheet.Service
	invoiceService *invoice.InvoiceService
}

func NewCollectionPipelineHandler(
	ingestService *ingest.IngestService,
	spreadsheetService *spreadsheet.Service,
	invoiceService *invoice.InvoiceService,
) *CollectionPipelineHandler {
	return &CollectionPipelineHandler{
		ingestService:  ingestService,
		spreadsheet:    spreadsheetService,
		invoiceService: invoiceService,
	}
}

func (h *CollectionPipelineHandler) ListDocuments(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	limit, err := strconv.Atoi(c.Query("limit", "100"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "limit must be a number")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "offset must be a number")
	}

	docs, err := h.ingestService.ListDocuments(
		ctx,
		userID,
		collectionID,
		c.Query("status"),
		limit,
		offset,
	)
	if err != nil {
		if errors.Is(err, dcollection.ErrCollectionNotFound) {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, docs, "documents retrieved")
}

func (h *CollectionPipelineHandler) ListHistory(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	limit, err := strconv.Atoi(c.Query("limit", "20"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "limit must be a number")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "offset must be a number")
	}

	history, err := h.ingestService.ListHistory(ctx, userID, collectionID, limit, offset)
	if err != nil {
		if errors.Is(err, dcollection.ErrCollectionNotFound) {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, history, "history retrieved")
}

func (h *CollectionPipelineHandler) ListHistoryItems(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}
	historyID := c.Params("historyId")
	if historyID == "" {
		return SendError(c, fiber.StatusBadRequest, "history id is required")
	}

	limit, err := strconv.Atoi(c.Query("limit", "100"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "limit must be a number")
	}
	offset, err := strconv.Atoi(c.Query("offset", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "offset must be a number")
	}

	items, err := h.ingestService.ListHistoryItems(
		ctx,
		userID,
		collectionID,
		historyID,
		c.Query("status"),
		limit,
		offset,
	)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, ingest.ErrHistoryNotFound):
			return SendError(c, fiber.StatusNotFound, "history not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, items, "history items retrieved")
}

func (h *CollectionPipelineHandler) GetDocumentInvoice(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}
	documentID := c.Params("documentId")
	if documentID == "" {
		return SendError(c, fiber.StatusBadRequest, "document id is required")
	}

	doc, err := h.ingestService.GetDocument(ctx, userID, collectionID, documentID)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, ingest.ErrDocumentNotFound):
			return SendError(c, fiber.StatusNotFound, "document not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	if doc.DocumentType != document.DocumentTypePDFInvoice {
		return SendError(c, fiber.StatusBadRequest, "invoice view only available for pdf_invoice")
	}

	inv, err := h.invoiceService.LoadInvoice(ctx, collectionID, doc.NormalizedRef)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "invoice normalized artifact not found")
	}

	return SendSuccess(c, fiber.StatusOK, inv, "invoice loaded")
}

func (h *CollectionPipelineHandler) GetSpreadsheetWorkbookMeta(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}
	documentID := c.Params("documentId")
	if documentID == "" {
		return SendError(c, fiber.StatusBadRequest, "document id is required")
	}

	meta, err := h.spreadsheet.GetWorkbookMeta(ctx, userID, collectionID, documentID)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, ingest.ErrDocumentNotFound):
			return SendError(c, fiber.StatusNotFound, "document not found")
		case errors.Is(err, spreadsheet.ErrUnsupportedDocumentType):
			return SendError(c, fiber.StatusBadRequest, "spreadsheet view only available for xlsx document types")
		case errors.Is(err, spreadsheet.ErrWorkbookArtifactMissing):
			return SendError(c, fiber.StatusNotFound, "workbook artifact not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, meta, "spreadsheet workbook metadata retrieved")
}

func (h *CollectionPipelineHandler) StreamSpreadsheetSheetRows(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}
	documentID := c.Params("documentId")
	if documentID == "" {
		return SendError(c, fiber.StatusBadRequest, "document id is required")
	}

	sheetName := c.Query("sheetName")
	if sheetName == "" {
		return SendError(c, fiber.StatusBadRequest, "sheetName is required")
	}

	headerRowNumber, err := strconv.Atoi(c.Query("headerRowNumber", "1"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "headerRowNumber must be a number")
	}
	startRow, err := strconv.Atoi(c.Query("startRow", "1"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "startRow must be a number")
	}
	maxRows, err := strconv.Atoi(c.Query("maxRows", "0"))
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "maxRows must be a number")
	}

	streamInput := spreadsheet.StreamSheetRowsInput{
		SheetName:       sheetName,
		HeaderRowNumber: headerRowNumber,
		StartRow:        startRow,
		MaxRows:         maxRows,
	}

	c.Set("Content-Type", "application/x-ndjson; charset=utf-8")
	c.Set("Cache-Control", "no-cache")
	c.Set("X-Accel-Buffering", "no")
	c.Status(fiber.StatusOK)

	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		writeLine := func(payload any) bool {
			b, marshalErr := json.Marshal(payload)
			if marshalErr != nil {
				return false
			}
			if _, writeErr := w.Write(append(b, '\n')); writeErr != nil {
				return false
			}
			return w.Flush() == nil
		}

		if !writeLine(map[string]any{
			"type":            "meta",
			"sheetName":       sheetName,
			"headerRowNumber": headerRowNumber,
			"startRow":        startRow,
			"maxRows":         maxRows,
		}) {
			return
		}

		streamErr := h.spreadsheet.StreamSheetRows(
			ctx,
			userID,
			collectionID,
			documentID,
			streamInput,
			func(row spreadsheet.SheetRow) error {
				if !writeLine(map[string]any{
					"type": "row",
					"row":  row,
				}) {
					return errors.New("stream write failed")
				}
				return nil
			},
		)
		if streamErr != nil {
			_ = writeLine(map[string]any{
				"type":    "error",
				"message": streamErr.Error(),
			})
			return
		}

		_ = writeLine(map[string]any{
			"type": "done",
		})
	})

	return nil
}
