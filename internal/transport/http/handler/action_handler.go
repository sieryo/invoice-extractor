package handler

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/sieryo/invoice-extractor/internal/app/action"
	dcollection "github.com/sieryo/invoice-extractor/internal/domain/collection"
	"github.com/sieryo/invoice-extractor/internal/domain/file"
)

type ActionHandler struct {
	actionService *action.Service
	fileStore     file.FileStore
}

func NewActionHandler(actionService *action.Service, fileStore file.FileStore) *ActionHandler {
	return &ActionHandler{
		actionService: actionService,
		fileStore:     fileStore,
	}
}

type RunActionRequest struct {
	ActionType             string          `json:"actionType"`
	Params                 json.RawMessage `json:"params,omitempty"`
	DocumentIDs            []string        `json:"documentIds,omitempty"`
	DocumentStatuses       []string        `json:"documentStatuses,omitempty"`
	IdempotencyKey         *string         `json:"idempotencyKey,omitempty"`
	RerunOfActionID        *string         `json:"rerunOfActionId,omitempty"`
	LegacyActionType       string          `json:"action_type,omitempty"`
	LegacyDocumentIDs      []string        `json:"document_ids,omitempty"`
	LegacyDocumentStatuses []string        `json:"document_statuses,omitempty"`
	LegacyIdempotencyKey   *string         `json:"idempotency_key,omitempty"`
	LegacyRerunOfActionID  *string         `json:"rerun_of_action_id,omitempty"`
}

func (h *ActionHandler) RunAction(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	var req RunActionRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	actionType := req.ActionType
	if actionType == "" {
		actionType = req.LegacyActionType
	}
	documentIDs := req.DocumentIDs
	if len(documentIDs) == 0 {
		documentIDs = req.LegacyDocumentIDs
	}
	documentStatuses := req.DocumentStatuses
	if len(documentStatuses) == 0 {
		documentStatuses = req.LegacyDocumentStatuses
	}
	idempotencyKey := req.IdempotencyKey
	if idempotencyKey == nil {
		idempotencyKey = req.LegacyIdempotencyKey
	}
	rerunOfActionID := req.RerunOfActionID
	if rerunOfActionID == nil {
		rerunOfActionID = req.LegacyRerunOfActionID
	}

	actionRecord, err := h.actionService.RunAction(ctx, action.RunRequest{
		UserID:           userID,
		CollectionID:     collectionID,
		ActionType:       actionType,
		Params:           req.Params,
		DocumentIDs:      documentIDs,
		DocumentStatuses: documentStatuses,
		IdempotencyKey:   idempotencyKey,
		RerunOfActionID:  rerunOfActionID,
	})
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		case errors.Is(err, action.ErrInvalidActionType):
			return SendError(c, fiber.StatusBadRequest, "action_type is required")
		case errors.Is(err, action.ErrInvalidDocumentIDs):
			return SendError(c, fiber.StatusBadRequest, "invalid document_ids")
		case errors.Is(err, action.ErrInvalidDocumentStatus):
			return SendError(c, fiber.StatusBadRequest, "invalid document_statuses filter")
		case errors.Is(err, action.ErrSnapshotDocStatus):
			return SendError(c, fiber.StatusBadRequest, "selected documents must be ready or warning")
		case errors.Is(err, action.ErrSnapshotDocNotFound):
			return SendError(c, fiber.StatusBadRequest, "some selected documents are not available")
		case errors.Is(err, action.ErrEmptySnapshot):
			return SendError(c, fiber.StatusBadRequest, "no documents matched snapshot filter")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusAccepted, actionRecord, "action queued")
}

func (h *ActionHandler) ListActions(c *fiber.Ctx) error {
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

	actions, listErr := h.actionService.ListActions(ctx, action.ListRequest{
		UserID:       userID,
		CollectionID: collectionID,
		Status:       c.Query("status"),
		Limit:        limit,
		Offset:       offset,
	})
	if listErr != nil {
		if errors.Is(listErr, dcollection.ErrCollectionNotFound) {
			return SendError(c, fiber.StatusNotFound, "collection not found")
		}
		return SendError(c, fiber.StatusInternalServerError, listErr.Error())
	}

	return SendSuccess(c, fiber.StatusOK, actions, "actions retrieved")
}

func (h *ActionHandler) GetActionDetail(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}
	actionID := c.Params("actionId")
	if actionID == "" {
		return SendError(c, fiber.StatusBadRequest, "action id is required")
	}

	detail, err := h.actionService.GetActionDetail(ctx, userID, collectionID, actionID)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, action.ErrActionNotFound):
			return SendError(c, fiber.StatusNotFound, "action not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, detail, "action detail retrieved")
}

func (h *ActionHandler) GetActionSpec(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	spec, err := h.actionService.GetActionSpec(ctx, userID, collectionID)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		case errors.Is(err, action.ErrSpecNotFound):
			return SendError(c, fiber.StatusNotFound, "action spec not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, spec, "action spec retrieved")
}

func (h *ActionHandler) DownloadActionOutput(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	actionID := c.Params("actionId")
	outputID := c.Params("outputId")
	if collectionID == "" || actionID == "" || outputID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection, action, and output id are required")
	}

	detail, err := h.actionService.GetActionDetail(ctx, userID, collectionID, actionID)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, action.ErrActionNotFound):
			return SendError(c, fiber.StatusNotFound, "action not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	var target *action.CollectionActionOutput
	for _, output := range detail.Outputs {
		if output != nil && output.ID == outputID {
			target = output
			break
		}
	}
	if target == nil {
		return SendError(c, fiber.StatusNotFound, "output not found")
	}
	if !strings.EqualFold(string(target.Kind), string(action.OutputKindFile)) {
		return SendError(c, fiber.StatusBadRequest, "output is not downloadable file")
	}

	name := filepath.Base(target.ObjectRef)
	data, readErr := h.fileStore.ReadArchive(ctx, collectionID, name)
	if readErr != nil {
		data, readErr = h.fileStore.Read(ctx, collectionID, name)
		if readErr != nil {
			return SendError(c, fiber.StatusNotFound, "output file not found")
		}
	}

	contentType := target.MimeType
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	filename := target.Name
	if filename == "" {
		filename = name
	}

	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	return c.Send(data)
}
