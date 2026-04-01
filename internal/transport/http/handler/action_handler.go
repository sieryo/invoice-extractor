package handler

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/google/uuid"
	"github.com/sieryo/invoice-extractor/internal/app/action"
	"github.com/sieryo/invoice-extractor/internal/app/document"
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
	ActionType       string          `json:"actionType"`
	Input            json.RawMessage `json:"input,omitempty"`
	DocumentIDs      []string        `json:"documentIds,omitempty"`
	DocumentStatuses []string        `json:"documentStatuses,omitempty"`
	IdempotencyKey   *string         `json:"idempotencyKey,omitempty"`
	RerunOfActionID  *string         `json:"rerunOfActionId,omitempty"`
}

type ResolveActionSpecRequest struct {
	ActionType  string   `json:"actionType"`
	Input       json.RawMessage `json:"input,omitempty"`
	DocumentIDs []string `json:"documentIds,omitempty"`
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

	actionRecord, err := h.actionService.RunAction(ctx, action.RunRequest{
		UserID:           userID,
		CollectionID:     collectionID,
		ActionType:       req.ActionType,
		Input:            req.Input,
		DocumentIDs:      req.DocumentIDs,
		DocumentStatuses: req.DocumentStatuses,
		IdempotencyKey:   req.IdempotencyKey,
		RerunOfActionID:  req.RerunOfActionID,
	})
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrCollectionFrozen):
			return SendError(c, fiber.StatusConflict, "collection sudah freeze dan tidak bisa menjalankan action baru")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		case errors.Is(err, action.ErrInvalidActionType):
			return SendError(c, fiber.StatusBadRequest, "actionType is required")
		case errors.Is(err, action.ErrActionNotSupported):
			return SendError(c, fiber.StatusBadRequest, "actionType is not available for this document type")
		case errors.Is(err, action.ErrActionDisabled):
			return SendError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, action.ErrActionRequirement):
			return SendError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, action.ErrInvalidActionParams):
			return SendError(c, fiber.StatusBadRequest, err.Error())
		case errors.Is(err, action.ErrInvalidActionSpec):
			return SendError(c, fiber.StatusBadRequest, "action spec is invalid")
		case errors.Is(err, action.ErrInvalidDocumentIDs):
			return SendError(c, fiber.StatusBadRequest, "invalid documentIds")
		case errors.Is(err, action.ErrInvalidDocumentStatus):
			return SendError(c, fiber.StatusBadRequest, "invalid documentStatuses filter")
		case errors.Is(err, action.ErrMinDocumentsRequired):
			return SendError(c, fiber.StatusBadRequest, "minimum selected documents for action is not met")
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

func (h *ActionHandler) ResolveActionSpec(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if collectionID == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	var req ResolveActionSpecRequest
	if err := c.BodyParser(&req); err != nil {
		return SendError(c, fiber.StatusBadRequest, "invalid request body")
	}

	spec, err := h.actionService.ResolveActionSpec(ctx, action.ResolveSpecRequest{
		UserID:       userID,
		CollectionID: collectionID,
		ActionType:   req.ActionType,
		Input:        req.Input,
		DocumentIDs:  req.DocumentIDs,
	})
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		case errors.Is(err, action.ErrSpecNotFound):
			return SendError(c, fiber.StatusNotFound, "action spec not found")
		case errors.Is(err, action.ErrActionNotSupported):
			return SendError(c, fiber.StatusBadRequest, "actionType is not available for this collection")
		case errors.Is(err, action.ErrInvalidDocumentIDs):
			return SendError(c, fiber.StatusBadRequest, "invalid documentIds")
		case errors.Is(err, action.ErrSnapshotDocNotFound):
			return SendError(c, fiber.StatusBadRequest, "some selected documents are not available")
		case errors.Is(err, action.ErrSnapshotDocStatus):
			return SendError(c, fiber.StatusBadRequest, "selected documents must be ready or warning")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	return SendSuccess(c, fiber.StatusOK, spec, "action spec resolved")
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

func (h *ActionHandler) DownloadActionArtifact(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := strings.TrimSpace(c.Params("id"))
	ref := strings.TrimSpace(c.Query("ref"))
	if collectionID == "" || ref == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id and ref are required")
	}
	if strings.Contains(ref, "..") || strings.HasPrefix(ref, "/") || strings.HasPrefix(ref, "\\") {
		return SendError(c, fiber.StatusBadRequest, "invalid artifact ref")
	}

	if _, err := h.actionService.GetActionSpec(ctx, userID, collectionID); err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	data, err := h.fileStore.ReadArchive(ctx, collectionID, ref)
	if err != nil {
		return SendError(c, fiber.StatusNotFound, "artifact file not found")
	}

	filename := filepath.Base(ref)
	if underscore := strings.Index(filename, "_"); underscore >= 0 && underscore < len(filename)-1 {
		filename = filename[underscore+1:]
	}
	contentType := http.DetectContentType(data)
	c.Set("Content-Type", contentType)
	c.Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
	return c.Send(data)
}

type UploadActionArtifactResponse struct {
	ActionType   string `json:"actionType"`
	ArtifactKey  string `json:"artifactKey"`
	Ref          string `json:"ref"`
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	SizeBytes    int64  `json:"sizeBytes"`
	Preview      any    `json:"preview,omitempty"`
}

func (h *ActionHandler) UploadActionArtifact(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := c.Params("id")
	if strings.TrimSpace(collectionID) == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id is required")
	}

	actionType := strings.TrimSpace(c.FormValue("actionType"))
	artifactKey := strings.TrimSpace(c.FormValue("artifactKey"))
	if actionType == "" || artifactKey == "" {
		return SendError(c, fiber.StatusBadRequest, "actionType and artifactKey are required")
	}

	docSpec, err := h.actionService.GetActionSpec(ctx, userID, collectionID)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrCollectionFrozen):
			return SendError(c, fiber.StatusConflict, "collection sudah freeze dan tidak bisa menjalankan action baru")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		case errors.Is(err, action.ErrSpecNotFound):
			return SendError(c, fiber.StatusNotFound, "action spec not found")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	actionSpec, found := docSpec.FindActionSpec(actionType)
	if !found {
		return SendError(c, fiber.StatusBadRequest, "action type is not available for this collection")
	}
	if !actionSpec.State.Enabled {
		reason := strings.TrimSpace(actionSpec.State.Message)
		if reason == "" {
			reason = "action tidak tersedia untuk collection ini"
		}
		return SendError(c, fiber.StatusConflict, reason)
	}

	artifactSpec, found := findArtifactInputSpec(actionSpec.ArtifactInputs, artifactKey)
	if !found {
		return SendError(c, fiber.StatusBadRequest, "artifact input key is not defined in action spec")
	}

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "file is required")
	}

	stream, err := fileHeader.Open()
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "failed to read file")
	}
	defer stream.Close()

	data, err := io.ReadAll(stream)
	if err != nil {
		return SendError(c, fiber.StatusBadRequest, "failed to read file content")
	}

	originalName := filepath.Base(strings.TrimSpace(fileHeader.Filename))
	if originalName == "" {
		originalName = "artifact_upload"
	}
	ext := strings.ToLower(filepath.Ext(originalName))

	mimeType := strings.ToLower(strings.TrimSpace(fileHeader.Header.Get("Content-Type")))
	if mimeType == "" && len(data) > 0 {
		sniff := data
		if len(sniff) > 512 {
			sniff = sniff[:512]
		}
		mimeType = strings.ToLower(http.DetectContentType(sniff))
	}

	if len(artifactSpec.AcceptExtensions) > 0 && !containsFolded(artifactSpec.AcceptExtensions, ext) {
		return SendError(c, fiber.StatusBadRequest, "file extension is not allowed for this artifact input")
	}
	if len(artifactSpec.AcceptMIMETypes) > 0 && mimeType != "" && !containsFolded(artifactSpec.AcceptMIMETypes, mimeType) {
		return SendError(c, fiber.StatusBadRequest, "file mime type is not allowed for this artifact input")
	}

	ref := filepath.ToSlash(filepath.Join(
		"inputs",
		sanitizeArtifactSegment(actionType),
		sanitizeArtifactSegment(artifactKey),
		uuid.NewString()+"_"+sanitizeArtifactFilename(originalName),
	))

	_, saveErr := h.fileStore.SaveArchive(ctx, collectionID, ref, data)
	if saveErr != nil {
		return SendError(c, fiber.StatusInternalServerError, "failed to save artifact file")
	}

	artifact, saveErr := h.actionService.SaveActionArtifact(
		ctx,
		userID,
		collectionID,
		actionType,
		artifactSpec,
		originalName,
		mimeType,
		int64(len(data)),
		ref,
		data,
	)
	if saveErr != nil {
		return SendError(c, fiber.StatusInternalServerError, saveErr.Error())
	}

	var preview any
	if len(artifact.PreviewJSON) > 0 {
		_ = json.Unmarshal(artifact.PreviewJSON, &preview)
	}

	return SendSuccess(c, fiber.StatusOK, UploadActionArtifactResponse{
		ActionType:   actionType,
		ArtifactKey:  artifactKey,
		Ref:          ref,
		OriginalName: originalName,
		MimeType:     mimeType,
		SizeBytes:    int64(len(data)),
		Preview:      preview,
	}, "artifact uploaded")
}

func (h *ActionHandler) ListLatestActionArtifacts(c *fiber.Ctx) error {
	ctx := c.Context()
	userID, ok := c.Locals("userId").(string)
	if !ok {
		return fiber.ErrUnauthorized
	}

	collectionID := strings.TrimSpace(c.Params("id"))
	actionType := strings.TrimSpace(c.Query("actionType"))
	if collectionID == "" || actionType == "" {
		return SendError(c, fiber.StatusBadRequest, "collection id and actionType are required")
	}

	items, err := h.actionService.GetLatestActionArtifacts(ctx, userID, collectionID, actionType)
	if err != nil {
		switch {
		case errors.Is(err, dcollection.ErrCollectionNotFound):
			return SendError(c, fiber.StatusNotFound, "collection not found")
		case errors.Is(err, dcollection.ErrInvalidNodeType):
			return SendError(c, fiber.StatusBadRequest, "target must be a typed collection")
		default:
			return SendError(c, fiber.StatusInternalServerError, err.Error())
		}
	}

	payload := make([]fiber.Map, 0, len(items))
	for _, item := range items {
		var preview any
		if len(item.PreviewJSON) > 0 {
			_ = json.Unmarshal(item.PreviewJSON, &preview)
		}
		payload = append(payload, fiber.Map{
			"id":           item.ID,
			"userId":       item.UserID,
			"collectionId": item.CollectionID,
			"actionType":   item.ActionType,
			"artifactKey":  item.ArtifactKey,
			"objectRef":    item.ObjectRef,
			"originalName": item.OriginalName,
			"mimeType":     item.MimeType,
			"sizeBytes":    item.SizeBytes,
			"preview":      preview,
			"createdAt":    item.CreatedAt,
			"updatedAt":    item.UpdatedAt,
		})
	}

	return SendSuccess(c, fiber.StatusOK, payload, "latest action artifacts retrieved")
}

func findArtifactInputSpec(
	inputs []document.ActionArtifactInputSpec,
	key string,
) (document.ActionArtifactInputSpec, bool) {
	target := strings.TrimSpace(key)
	if target == "" {
		return document.ActionArtifactInputSpec{}, false
	}
	for _, item := range inputs {
		if strings.EqualFold(strings.TrimSpace(item.Key), target) {
			return item, true
		}
	}
	return document.ActionArtifactInputSpec{}, false
}

func containsFolded(items []string, value string) bool {
	target := strings.ToLower(strings.TrimSpace(value))
	if target == "" {
		return false
	}
	for _, item := range items {
		if strings.EqualFold(strings.TrimSpace(item), target) {
			return true
		}
	}
	return false
}

func sanitizeArtifactSegment(raw string) string {
	segment := strings.TrimSpace(strings.ToLower(raw))
	if segment == "" {
		return "unknown"
	}

	b := strings.Builder{}
	for _, r := range segment {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		case r == '_' || r == '-':
			b.WriteRune(r)
		default:
			b.WriteRune('_')
		}
	}
	out := strings.Trim(b.String(), "_-")
	if out == "" {
		return "unknown"
	}
	return out
}

func sanitizeArtifactFilename(raw string) string {
	name := strings.TrimSpace(filepath.Base(raw))
	if name == "" {
		return "artifact.txt"
	}

	name = strings.ReplaceAll(name, "\\", "_")
	name = strings.ReplaceAll(name, "/", "_")
	name = strings.ReplaceAll(name, ":", "_")
	name = strings.ReplaceAll(name, "..", "_")

	if strings.TrimSpace(name) == "" {
		return "artifact.txt"
	}

	// If a control character sneaks in, replace with underscore.
	buf := bytes.NewBuffer(nil)
	for i := 0; i < len(name); i++ {
		ch := name[i]
		if ch < 32 || ch == 127 {
			buf.WriteByte('_')
			continue
		}
		buf.WriteByte(ch)
	}
	return buf.String()
}
