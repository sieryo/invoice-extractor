package handler

import (
	"io"
	"strconv"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/ingest"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
)

type UploadSessionHandler struct {
	ingestService *ingest.IngestService
}

func NewUploadSessionHandler(ingestService *ingest.IngestService) *UploadSessionHandler {
	return &UploadSessionHandler{
		ingestService: ingestService,
	}
}

type StartUploadSessionRequest struct {
	ClientSessionKey       *string `json:"clientSessionKey,omitempty"`
	LegacyClientSessionKey *string `json:"client_session_key,omitempty"`
}

type ResolveUploadDuplicatesRequest struct {
	Decision string `json:"decision"`
}

func (h *UploadSessionHandler) StartSession(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection_id is required")
	}

	var req StartUploadSessionRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	clientSessionKey := req.ClientSessionKey
	if clientSessionKey == nil {
		clientSessionKey = req.LegacyClientSessionKey
	}

	session, err := h.ingestService.CreateSession(ctx, userID, collectionID, clientSessionKey)
	if err != nil {
		switch {
		case err == dcollection.ErrCollectionNotFound:
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case err == dcollection.ErrInvalidCollectionKind:
			return SendError(c, fiber.StatusBadRequest, "jenis collection ini sedang dinonaktifkan")
		case err == dcollection.ErrCollectionFrozen:
			return SendError(c, fiber.StatusConflict, "collection sudah freeze dan tidak bisa menerima dokumen baru")
		default:
			return SendError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusCreated, session, "upload session created")
}

func (h *UploadSessionHandler) UploadChunk(c *fiber.Ctx) error {
	ctx := c.Context()

	sessionID := c.Params("id")
	if sessionID == "" {
		return SendError(c, fiber.StatusBadRequest, "session_id is required")
	}

	chunkIndex, err := strconv.Atoi(c.FormValue("chunk_index", "0"))
	if err != nil || chunkIndex < 0 {
		return SendError(c, fiber.StatusBadRequest, "chunk_index must be a non-negative integer")
	}

	sourceOrderStart, err := strconv.Atoi(c.FormValue("source_order_start", "0"))
	if err != nil || sourceOrderStart < 0 {
		return SendError(c, fiber.StatusBadRequest, "source_order_start must be a non-negative integer")
	}

	idempotencyKey := c.FormValue("idempotency_key")
	if idempotencyKey == "" {
		return SendError(c, fiber.StatusBadRequest, "idempotency_key is required")
	}

	var requestChecksum *string
	if v := c.FormValue("request_checksum"); v != "" {
		requestChecksum = &v
	}

	form, err := c.MultipartForm()
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid multipart form")
	}

	rawFiles := form.File["files"]
	if len(rawFiles) == 0 {
		return SendError(c, fiber.StatusBadRequest, "no files uploaded")
	}

	files := make([]ingest.SourceUploadFile, 0, len(rawFiles))
	for _, f := range rawFiles {
		src, err := f.Open()
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "failed to open file")
		}

		b, err := io.ReadAll(src)
		src.Close()
		if err != nil {
			return SendError(c, fiber.StatusInternalServerError, "failed to read file")
		}

		files = append(files, ingest.SourceUploadFile{
			Name: f.Filename,
			Data: b,
		})
	}

	chunk, err := h.ingestService.UploadChunk(ctx, sessionID, ingest.ChunkUploadInput{
		ChunkIndex:      chunkIndex,
		IdempotencyKey:  idempotencyKey,
		RequestChecksum: requestChecksum,
	}, files, sourceOrderStart)
	if err != nil {
		switch {
		case err == dcollection.ErrCollectionFrozen:
			return SendError(c, fiber.StatusConflict, "collection sudah freeze dan tidak bisa menerima dokumen baru")
		default:
			return SendError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, chunk, "chunk uploaded")
}

func (h *UploadSessionHandler) FinalizeSession(c *fiber.Ctx) error {
	ctx := c.Context()

	sessionID := c.Params("id")
	if sessionID == "" {
		return SendError(c, fiber.StatusBadRequest, "session_id is required")
	}

	session, err := h.ingestService.FinalizeSession(ctx, sessionID)
	if err != nil {
		switch {
		case err == dcollection.ErrCollectionFrozen:
			return SendError(c, fiber.StatusConflict, "collection sudah freeze dan tidak bisa menerima dokumen baru")
		default:
			return SendError(c, fiber.StatusBadRequest, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, session, "upload session finalized")
}

func (h *UploadSessionHandler) GetSession(c *fiber.Ctx) error {
	ctx := c.Context()

	sessionID := c.Params("id")
	if sessionID == "" {
		return SendError(c, fiber.StatusBadRequest, "session_id is required")
	}

	detail, err := h.ingestService.GetSessionDetail(ctx, sessionID)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, detail, "upload session retrieved")
}

func (h *UploadSessionHandler) ResolveDuplicates(c *fiber.Ctx) error {
	ctx := c.Context()

	userID, ok := c.Locals("userId").(string)
	if !ok {
		return SendError(c, fiber.StatusUnauthorized, "unauthorized")
	}

	sessionID := c.Params("id")
	if sessionID == "" {
		return SendError(c, fiber.StatusBadRequest, "session_id is required")
	}

	var req ResolveUploadDuplicatesRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	session, err := h.ingestService.ResolvePendingDuplicates(
		ctx,
		userID,
		sessionID,
		ingest.ResolveDuplicateDecision(req.Decision),
	)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, err.Error())
	}

	return SendSuccess(c, fiber.StatusOK, session, "duplicate resolution applied")
}
