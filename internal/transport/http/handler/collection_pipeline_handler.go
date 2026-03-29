package handler

import (
	"errors"
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/document"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
	"github.com/sieryo/invoice-extractor/internal/app/invoice"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type CollectionPipelineHandler struct {
	ingestService  *ingest.IngestService
	invoiceService *invoice.InvoiceService
}

func NewCollectionPipelineHandler(
	ingestService *ingest.IngestService,
	invoiceService *invoice.InvoiceService,
) *CollectionPipelineHandler {
	return &CollectionPipelineHandler{
		ingestService:  ingestService,
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

	if doc.CollectionKind != document.CollectionKindInvoiceCompany {
		return SendError(c, fiber.StatusBadRequest, "invoice view only available for invoice_company")
	}

	inv, err := h.invoiceService.LoadInvoice(ctx, collectionID, doc.NormalizedRef)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "invoice normalized artifact not found")
	}

	return SendSuccess(c, fiber.StatusOK, inv, "invoice loaded")
}

func (h *CollectionPipelineHandler) ReplaceDocumentSource(c *fiber.Ctx) error {
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

	form, err := c.MultipartForm()
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid multipart form")
	}
	rawFiles := form.File["file"]
	if len(rawFiles) == 0 {
		return SendError(c, fiber.StatusBadRequest, "file is required")
	}
	if len(rawFiles) > 1 {
		return SendError(c, fiber.StatusBadRequest, "only one file is allowed")
	}

	src, err := rawFiles[0].Open()
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to open file")
	}
	defer src.Close()

	b, err := io.ReadAll(src)
	if err != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to read file")
	}

	doc, err := h.ingestService.ReplaceDocumentSource(
		ctx,
		userID,
		collectionID,
		documentID,
		rawFiles[0].Filename,
		b,
	)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, ingest.ErrDocumentNotFound):
			return SendError(c, fiber.StatusNotFound, "document not found")
		case errors.Is(err, ingest.ErrReplaceSourceNotSupported):
			return SendError(c, fiber.StatusBadRequest, "replace source hanya tersedia untuk dokumen excel")
		case errors.Is(err, dcollection.ErrCollectionFrozen):
			return SendError(c, fiber.StatusConflict, "collection sudah freeze dan tidak bisa mengubah dokumen")
		default:
			return SendError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, doc, "document source replaced")
}
