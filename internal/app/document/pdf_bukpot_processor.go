package document

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/sieryo/invoice-extractor/internal/app/bukpot"
	bukpotdomain "github.com/sieryo/invoice-extractor/internal/domain/bukpot"
	dfile "github.com/sieryo/invoice-extractor/internal/domain/file"
)

type PDFBukpotProcessor struct {
	docType    DocumentType
	forcedKind bukpotdomain.Kind
	service    *bukpot.Service
	fileStore  dfile.FileStore
	actions    map[string]bukpotActionHandler
}

const (
	bukpotRenameByCategoryActionType = "rename_by_category"
	bukpotRenameActionType           = "rename_bukpot"
	defaultBukpotNameTemplate        = "{{nomorBuktiPotong}} - {{namaPenerima}}"
	bukpotUnknownCategory            = "UNKNOWN"
	bukpotDefaultFallbackFilename    = "bukpot"
)

var bukpotTemplateTokenRegex = regexp.MustCompile(`\{\{\s*([a-zA-Z0-9_]+)\s*\}\}`)
var bukpotInvalidFilenameCharRegex = regexp.MustCompile(`[<>:"/\\|?*\x00-\x1F]`)

type bukpotRenameParams struct {
	FilenameTemplate string `json:"filenameTemplate"`
}

type bukpotActionHandler func(ctx context.Context, req ActionRequest) (ActionResult, error)

type bukpotZipEntry struct {
	Category string
	Name     string
	Data     []byte
}

type bukpotNormalizedPayload struct {
	SourceName  string                       `json:"source_name"`
	DocumentTag string                       `json:"document_tag"`
	Bukpot      *bukpotdomain.ParsedDocument `json:"bukpot"`
}

func NewPDFBukpotProcessor(
	docType DocumentType,
	service *bukpot.Service,
	fileStore dfile.FileStore,
) *PDFBukpotProcessor {
	p := &PDFBukpotProcessor{
		docType:    docType,
		forcedKind: mapBukpotKindFromDocType(docType),
		service:    service,
		fileStore:  fileStore,
		actions:    map[string]bukpotActionHandler{},
	}

	p.registerActionHandler(bukpotRenameActionType, p.runRenameWithTemplate)
	if supportsBukpotCategoryAction(docType) {
		p.registerActionHandler(bukpotRenameByCategoryActionType, p.runRenameByCategory)
	}

	return p
}

func (p *PDFBukpotProcessor) Type() DocumentType {
	return p.docType
}

func (p *PDFBukpotProcessor) Ingest(ctx context.Context, req IngestRequest) (IngestResult, error) {
	startedAt := time.Now()
	result := IngestResult{
		BatchID:      req.RequestID,
		CollectionID: req.CollectionID,
		DocumentType: string(req.DocumentType),
		Items:        make([]IngestItemResult, 0, len(req.Sources)),
		StartedAt:    startedAt,
	}

	if !p.forcedKind.IsValid() {
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: unsupported bukpot document type %s", ErrProcessorNotImplemented, p.docType)
	}

	for _, source := range req.Sources {
		item := p.ingestSource(ctx, req, source)
		result.Items = append(result.Items, item)
	}

	for _, item := range result.Items {
		switch item.Status {
		case IngestStatusReady:
			result.Success++
		case IngestStatusWarning:
			result.Warning++
		case IngestStatusDuplicate:
			result.Duplicate++
		default:
			result.Failed++
		}
	}

	result.Total = len(result.Items)
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *PDFBukpotProcessor) RunAction(ctx context.Context, req ActionRequest) (ActionResult, error) {
	actionType := strings.ToLower(strings.TrimSpace(req.ActionType))
	handler, ok := p.actions[actionType]
	if !ok {
		result := ActionResult{
			ActionID:    req.ActionID,
			ActionType:  req.ActionType,
			StartedAt:   req.RequestedAt,
			Outputs:     []ActionOutput{},
			ItemResults: []ActionItemResult{},
			Status:      "failed",
			Message:     fmt.Sprintf("unsupported action %s for %s", req.ActionType, p.docType),
			FinishedAt:  time.Now(),
		}
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.docType)
	}

	return handler(ctx, req)
}

func (p *PDFBukpotProcessor) registerActionHandler(
	actionType string,
	handler bukpotActionHandler,
) {
	normalizedType := strings.ToLower(strings.TrimSpace(actionType))
	if normalizedType == "" || handler == nil {
		return
	}
	p.actions[normalizedType] = handler
}

func (p *PDFBukpotProcessor) runRenameByCategory(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if !supportsBukpotCategoryAction(p.docType) {
		result.Status = "failed"
		result.Message = "rename_by_category is only available for BPPU and BP21"
		result.FinishedAt = time.Now()
		return result, fmt.Errorf("%w: action %s for %s", ErrProcessorNotImplemented, req.ActionType, p.docType)
	}

	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, errors.New("snapshot is empty")
	}

	entries := make([]bukpotZipEntry, 0, len(req.SnapshotDocs))
	usedNamesByCategory := make(map[string]map[string]struct{}, len(req.SnapshotDocs))

	for _, doc := range req.SnapshotDocs {
		payload, rawBytes, loadErr := p.loadBukpotSourceForAction(ctx, req.CollectionID, doc)
		if loadErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: failed to prepare source", doc.SourceName),
				Error:      loadErr.Error(),
			})
			continue
		}

		category, dokumenNomor := extractBukpotCategoryAndDocumentNumber(p.docType, payload.Bukpot)
		category = sanitizeBukpotPathSegment(category)
		if category == "" {
			category = bukpotUnknownCategory
		}

		filename := renderBukpotDocumentNumberFilename(dokumenNomor, payload.Bukpot, payload.SourceName, doc.SourceName)
		filename = sanitizeBukpotFilename(filename)
		if filename == "" {
			filename = bukpotDefaultFallbackFilename
		}
		if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
			filename += ".pdf"
		}

		if _, ok := usedNamesByCategory[category]; !ok {
			usedNamesByCategory[category] = map[string]struct{}{}
		}
		filename = ensureUniqueBukpotFilename(filename, usedNamesByCategory[category])

		result.ItemResults = append(result.ItemResults, ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     "success",
			Message:    fmt.Sprintf("renamed to %s/%s", category, filename),
		})

		entries = append(entries, bukpotZipEntry{
			Category: category,
			Name:     filename,
			Data:     rawBytes,
		})
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)
	if result.Success == 0 {
		result.Status = "failed"
		result.Message = "rename failed for all selected documents"
		result.FinishedAt = time.Now()
		return result, errors.New("no bukpot document renamed successfully")
	}

	zipBytes, zipErr := buildBukpotZipArchive(entries, true)
	if zipErr != nil {
		result.Status = "failed"
		result.Message = "failed to build zip output"
		result.FinishedAt = time.Now()
		return result, zipErr
	}

	if err := p.attachBukpotZipOutput(ctx, req.CollectionID, zipBytes, p.docType, bukpotRenameByCategoryActionType, &result); err != nil {
		result.Status = "failed"
		result.Message = "failed to save zip output"
		result.FinishedAt = time.Now()
		return result, err
	}

	if result.Failed > 0 {
		result.Status = "partial"
		result.Message = fmt.Sprintf("rename completed with partial results (%d success, %d failed)", result.Success, result.Failed)
	} else {
		result.Status = "success"
		result.Message = "rename completed"
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *PDFBukpotProcessor) runRenameWithTemplate(ctx context.Context, req ActionRequest) (ActionResult, error) {
	result := ActionResult{
		ActionID:    req.ActionID,
		ActionType:  req.ActionType,
		StartedAt:   req.RequestedAt,
		ItemResults: make([]ActionItemResult, 0, len(req.SnapshotDocs)),
		Outputs:     make([]ActionOutput, 0, 1),
	}

	if len(req.SnapshotDocs) == 0 {
		result.Status = "failed"
		result.Message = "snapshot is empty"
		result.FinishedAt = time.Now()
		return result, errors.New("snapshot is empty")
	}

	params, err := parseBukpotRenameParams(req.Params)
	if err != nil {
		result.Status = "failed"
		result.Message = "invalid action params"
		result.FinishedAt = time.Now()
		return result, err
	}

	entries := make([]bukpotZipEntry, 0, len(req.SnapshotDocs))
	usedNames := map[string]struct{}{}
	hasWarning := false

	for _, doc := range req.SnapshotDocs {
		payload, rawBytes, loadErr := p.loadBukpotSourceForAction(ctx, req.CollectionID, doc)
		if loadErr != nil {
			result.ItemResults = append(result.ItemResults, ActionItemResult{
				DocumentID: doc.DocumentID,
				Status:     "failed",
				Message:    fmt.Sprintf("%s: failed to prepare source", doc.SourceName),
				Error:      loadErr.Error(),
			})
			continue
		}

		values := buildBukpotTemplateMap(p.docType, payload.Bukpot, payload.SourceName, payload.DocumentTag)
		filename, warnings := renderBukpotFilename(params.FilenameTemplate, values)
		filename = ensureUniqueBukpotFilename(filename, usedNames)

		itemStatus := "success"
		itemMessage := fmt.Sprintf("renamed to %s", filename)
		if len(warnings) > 0 {
			itemStatus = "warning"
			itemMessage = fmt.Sprintf("renamed with warnings to %s", filename)
			hasWarning = true
		}

		result.ItemResults = append(result.ItemResults, ActionItemResult{
			DocumentID: doc.DocumentID,
			Status:     itemStatus,
			Message:    itemMessage,
			Warnings:   warnings,
		})

		entries = append(entries, bukpotZipEntry{
			Name: filename,
			Data: rawBytes,
		})
	}

	result.Total = len(req.SnapshotDocs)
	result.Success, result.Warning, result.Failed, result.Skipped = summarizeActionItems(result.ItemResults)
	if result.Success == 0 && result.Warning == 0 {
		result.Status = "failed"
		result.Message = "rename failed for all selected documents"
		result.FinishedAt = time.Now()
		return result, errors.New("no bukpot document renamed successfully")
	}

	zipBytes, zipErr := buildBukpotZipArchive(entries, false)
	if zipErr != nil {
		result.Status = "failed"
		result.Message = "failed to build zip output"
		result.FinishedAt = time.Now()
		return result, zipErr
	}

	if err := p.attachBukpotZipOutput(ctx, req.CollectionID, zipBytes, p.docType, bukpotRenameActionType, &result); err != nil {
		result.Status = "failed"
		result.Message = "failed to save zip output"
		result.FinishedAt = time.Now()
		return result, err
	}

	if result.Failed > 0 {
		result.Status = "partial"
		result.Message = fmt.Sprintf(
			"rename completed with partial results (%d success, %d warning, %d failed)",
			result.Success,
			result.Warning,
			result.Failed,
		)
	} else if hasWarning {
		result.Status = "warning"
		result.Message = "rename completed with warnings"
	} else {
		result.Status = "success"
		result.Message = "rename completed"
	}
	result.FinishedAt = time.Now()
	return result, nil
}

func (p *PDFBukpotProcessor) ingestSource(
	ctx context.Context,
	req IngestRequest,
	source IngestSource,
) IngestItemResult {
	forcedKind := p.forcedKind
	parsed, err := p.service.ParseFile(ctx, bukpotdomain.FileInput{
		UploadIndex: source.SourceOrder,
		SourceName:  source.OriginalName,
		Path:        source.TempPath,
	}, &forcedKind)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{err.Error()},
		}
	}

	if parsed == nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{"empty parser response"},
		}
	}

	if parsed.Error != nil && strings.TrimSpace(*parsed.Error) != "" {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{strings.TrimSpace(*parsed.Error)},
		}
	}

	if parsed.Data == nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to parse bukpot",
			Errors:       []string{"parsed document is empty"},
		}
	}

	artifacts := make([]Artifact, 0, 2)

	if req.Policy.KeepRaw {
		rawBytes, readErr := os.ReadFile(source.TempPath)
		if readErr != nil {
			return IngestItemResult{
				SourceID:     source.SourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to read bukpot source file",
				Errors:       []string{readErr.Error()},
			}
		}

		rawName := source.SourceID + filepath.Ext(source.OriginalName)
		if writeErr := p.fileStore.WriteFile(ctx, req.CollectionID, rawName, rawBytes); writeErr != nil {
			return IngestItemResult{
				SourceID:     source.SourceID,
				OriginalName: source.OriginalName,
				SHA256:       source.SHA256,
				Status:       IngestStatusFailed,
				Message:      "failed to persist bukpot raw file",
				Errors:       []string{writeErr.Error()},
			}
		}

		artifacts = append(artifacts, Artifact{
			Kind:     "raw",
			ObjectID: rawName,
			MimeType: "application/pdf",
			Size:     int64(len(rawBytes)),
		})
	}

	normalizedPayload := map[string]any{
		"source_id":     source.SourceID,
		"source_name":   source.OriginalName,
		"source_sha256": source.SHA256,
		"document_type": string(req.DocumentType),
		"document_tag":  strings.TrimSpace(parsed.Data.DocumentTag),
		"bukpot":        parsed.Data,
		"processed_at":  time.Now().UTC(),
	}
	normalizedBytes, err := json.Marshal(normalizedPayload)
	if err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to encode bukpot normalized payload",
			Errors:       []string{err.Error()},
		}
	}

	normalizedName := fmt.Sprintf("normalized_%s.json", source.SourceID)
	if err := p.fileStore.WriteFile(ctx, req.CollectionID, normalizedName, normalizedBytes); err != nil {
		return IngestItemResult{
			SourceID:     source.SourceID,
			OriginalName: source.OriginalName,
			SHA256:       source.SHA256,
			Status:       IngestStatusFailed,
			Message:      "failed to persist bukpot normalized payload",
			Errors:       []string{err.Error()},
		}
	}

	artifacts = append(artifacts, Artifact{
		Kind:     "normalized",
		ObjectID: normalizedName,
		MimeType: "application/json",
		Size:     int64(len(normalizedBytes)),
	})

	return IngestItemResult{
		SourceID:     source.SourceID,
		OriginalName: source.OriginalName,
		SHA256:       source.SHA256,
		DocumentTag:  strings.TrimSpace(parsed.Data.DocumentTag),
		Status:       IngestStatusReady,
		Message:      "bukpot parsed",
		Artifacts:    artifacts,
	}
}

func mapBukpotKindFromDocType(docType DocumentType) bukpotdomain.Kind {
	switch docType {
	case DocumentTypePDFBukpotBPPU:
		return bukpotdomain.KindBPPU
	case DocumentTypePDFBukpotBP21:
		return bukpotdomain.KindBP21
	case DocumentTypePDFBukpotBPA1:
		return bukpotdomain.KindBPA1
	default:
		return ""
	}
}

func supportsBukpotCategoryAction(docType DocumentType) bool {
	return docType == DocumentTypePDFBukpotBPPU || docType == DocumentTypePDFBukpotBP21
}

func parseBukpotRenameParams(raw json.RawMessage) (bukpotRenameParams, error) {
	params := bukpotRenameParams{FilenameTemplate: defaultBukpotNameTemplate}
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || strings.EqualFold(trimmed, "null") {
		return params, nil
	}
	if err := json.Unmarshal(raw, &params); err != nil {
		return params, fmt.Errorf("invalid rename params: %w", err)
	}
	params.FilenameTemplate = strings.TrimSpace(params.FilenameTemplate)
	if params.FilenameTemplate == "" {
		params.FilenameTemplate = defaultBukpotNameTemplate
	}
	return params, nil
}

func (p *PDFBukpotProcessor) loadBukpotSourceForAction(
	ctx context.Context,
	collectionID string,
	doc ActionSnapshotDocument,
) (*bukpotNormalizedPayload, []byte, error) {
	if strings.TrimSpace(doc.NormalizedRef) == "" {
		return nil, nil, errors.New("normalized artifact is missing")
	}
	if strings.TrimSpace(doc.RawRef) == "" {
		return nil, nil, errors.New("raw artifact is missing")
	}

	payload, err := p.loadBukpotNormalizedPayload(ctx, collectionID, doc.NormalizedRef)
	if err != nil || payload == nil || payload.Bukpot == nil {
		return nil, nil, errors.New(errorText(err, "invalid normalized bukpot payload"))
	}

	rawBytes, readErr := p.fileStore.Read(ctx, collectionID, doc.RawRef)
	if readErr != nil {
		return nil, nil, readErr
	}

	if strings.TrimSpace(payload.SourceName) == "" {
		payload.SourceName = doc.SourceName
	}

	return payload, rawBytes, nil
}

func (p *PDFBukpotProcessor) loadBukpotNormalizedPayload(
	ctx context.Context,
	collectionID string,
	ref string,
) (*bukpotNormalizedPayload, error) {
	b, err := p.fileStore.Read(ctx, collectionID, ref)
	if err != nil {
		return nil, err
	}

	var payload bukpotNormalizedPayload
	if err := json.Unmarshal(b, &payload); err == nil && payload.Bukpot != nil {
		return &payload, nil
	}

	var parsed bukpotdomain.ParsedDocument
	if err := json.Unmarshal(b, &parsed); err == nil {
		return &bukpotNormalizedPayload{Bukpot: &parsed}, nil
	}

	return nil, errors.New("invalid normalized bukpot payload")
}

func extractBukpotCategoryAndDocumentNumber(
	docType DocumentType,
	parsed *bukpotdomain.ParsedDocument,
) (string, string) {
	if parsed == nil {
		return bukpotUnknownCategory, ""
	}

	switch docType {
	case DocumentTypePDFBukpotBPPU:
		if parsed.BPPU == nil {
			return bukpotUnknownCategory, ""
		}
		return extractCategoryAndDocumentNumberFromReference(parsed.BPPU.DokumenReferensiNomor)
	case DocumentTypePDFBukpotBP21:
		if parsed.BP21 == nil {
			return bukpotUnknownCategory, ""
		}
		return extractCategoryAndDocumentNumberFromReference(parsed.BP21.DokumenReferensiNomor)
	default:
		return bukpotUnknownCategory, ""
	}
}

func extractCategoryAndDocumentNumberFromReference(raw string) (category string, numberedPrefix string) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return bukpotUnknownCategory, ""
	}

	parts := strings.SplitN(trimmed, " - ", 2)
	if len(parts) == 1 {
		return bukpotUnknownCategory, strings.TrimSpace(trimmed)
	}

	numberedPrefix = strings.TrimSpace(parts[0])
	category = strings.TrimSpace(parts[1])
	if category == "" {
		category = bukpotUnknownCategory
	}
	return category, numberedPrefix
}

func renderBukpotDocumentNumberFilename(
	numberedPrefix string,
	parsed *bukpotdomain.ParsedDocument,
	sourceName string,
	fallbackSourceName string,
) string {
	documentNumber := strings.TrimSpace(numberedPrefix)
	receiver := strings.TrimSpace(getBukpotReceiverName(parsed))
	if documentNumber == "" {
		return renderBukpotFallbackFilename(parsed, sourceName, fallbackSourceName)
	}
	if receiver == "" {
		return documentNumber
	}
	return documentNumber + " - " + receiver
}

func renderBukpotFallbackFilename(
	parsed *bukpotdomain.ParsedDocument,
	sourceName string,
	fallbackSourceName string,
) string {
	nomor := strings.TrimSpace(getBukpotNomorBukti(parsed))
	nama := strings.TrimSpace(getBukpotReceiverName(parsed))
	switch {
	case nomor != "" && nama != "":
		return nomor + " - " + nama
	case nomor != "":
		return nomor
	case nama != "":
		return nama
	}

	normalizedSource := strings.TrimSuffix(strings.TrimSpace(sourceName), filepath.Ext(sourceName))
	if normalizedSource != "" {
		return normalizedSource
	}
	normalizedFallback := strings.TrimSuffix(strings.TrimSpace(fallbackSourceName), filepath.Ext(fallbackSourceName))
	if normalizedFallback != "" {
		return normalizedFallback
	}
	return bukpotDefaultFallbackFilename
}

func getBukpotReceiverName(parsed *bukpotdomain.ParsedDocument) string {
	if parsed == nil {
		return ""
	}
	switch parsed.Kind {
	case bukpotdomain.KindBPPU:
		if parsed.BPPU != nil {
			return parsed.BPPU.NamaPenerima
		}
	case bukpotdomain.KindBP21:
		if parsed.BP21 != nil {
			return parsed.BP21.NamaPenerima
		}
	case bukpotdomain.KindBPA1:
		if parsed.BPA1 != nil {
			return parsed.BPA1.NamaPenerima
		}
	}
	return ""
}

func getBukpotNomorBukti(parsed *bukpotdomain.ParsedDocument) string {
	if parsed == nil {
		return ""
	}
	switch parsed.Kind {
	case bukpotdomain.KindBPPU:
		if parsed.BPPU != nil {
			return parsed.BPPU.NomorBuktiPotong
		}
	case bukpotdomain.KindBP21:
		if parsed.BP21 != nil {
			return parsed.BP21.NomorBuktiPotong
		}
	case bukpotdomain.KindBPA1:
		if parsed.BPA1 != nil {
			return parsed.BPA1.NomorBuktiPotong
		}
	}
	return ""
}

func buildBukpotTemplateMap(
	docType DocumentType,
	parsed *bukpotdomain.ParsedDocument,
	sourceName string,
	documentTag string,
) map[string]string {
	values := map[string]string{}

	put := func(key string, value string) {
		trimmed := strings.TrimSpace(value)
		if trimmed == "" {
			return
		}
		values[normalizeBukpotTemplateToken(key)] = trimmed
	}

	put("sourceName", strings.TrimSuffix(strings.TrimSpace(sourceName), filepath.Ext(sourceName)))
	put("documentTag", documentTag)
	put("documentType", string(docType))

	if parsed == nil {
		return values
	}

	put("kind", parsed.Kind.String())
	put("nomorBuktiPotong", getBukpotNomorBukti(parsed))
	put("namaPenerima", getBukpotReceiverName(parsed))

	switch docType {
	case DocumentTypePDFBukpotBPPU:
		if parsed.BPPU == nil {
			return values
		}
		put("dokumenReferensiNomor", parsed.BPPU.DokumenReferensiNomor)
		put("masaPajak", parsed.BPPU.MasaPajak)
		put("sifatPemotongan", parsed.BPPU.SifatPemotongan)
		put("statusBukti", parsed.BPPU.StatusBukti)
		put("npwpNikPenerima", parsed.BPPU.NPWPNIKPenerima)
		put("namaPemotong", parsed.BPPU.NamaPemotong)
		put("npwpNikPemotong", parsed.BPPU.NPWPNIKPemotong)
		put("dokumenReferensiJenis", parsed.BPPU.DokumenReferensiJenis)
		put("dokumenReferensiTanggal", parsed.BPPU.DokumenReferensiTanggal)
	case DocumentTypePDFBukpotBP21:
		if parsed.BP21 == nil {
			return values
		}
		put("dokumenReferensiNomor", parsed.BP21.DokumenReferensiNomor)
		put("masaPajak", parsed.BP21.MasaPajak)
		put("sifatPemotongan", parsed.BP21.SifatPemotongan)
		put("statusBukti", parsed.BP21.StatusBukti)
		put("npwpNikPenerima", parsed.BP21.NIKNPWPPenerima)
		put("namaPemotong", parsed.BP21.NamaPemotong)
		put("npwpNikPemotong", parsed.BP21.NPWPNIKPemotong)
		put("dokumenReferensiJenis", parsed.BP21.DokumenReferensiJenis)
		put("dokumenReferensiTanggal", parsed.BP21.DokumenReferensiTanggal)
	case DocumentTypePDFBukpotBPA1:
		if parsed.BPA1 == nil {
			return values
		}
		put("periodePenghasilan", parsed.BPA1.PeriodePenghasilan)
		put("sifatPemotongan", parsed.BPA1.SifatPemotongan)
		put("statusBukti", parsed.BPA1.StatusBukti)
		put("npwpNikPenerima", parsed.BPA1.NIKNPWPPenerima)
		put("posisi", parsed.BPA1.Posisi)
		put("statusPtkp", parsed.BPA1.StatusPTKP)
		put("namaPemotong", parsed.BPA1.NamaPemotong)
		put("npwpNikPemotong", parsed.BPA1.NPWPNIKPemotong)
	}

	return values
}

func renderBukpotFilename(template string, values map[string]string) (string, []string) {
	resolvedTemplate := strings.TrimSpace(template)
	if resolvedTemplate == "" {
		resolvedTemplate = defaultBukpotNameTemplate
	}

	missingTokens := make([]string, 0, 2)
	rendered := bukpotTemplateTokenRegex.ReplaceAllStringFunc(resolvedTemplate, func(match string) string {
		sub := bukpotTemplateTokenRegex.FindStringSubmatch(match)
		if len(sub) < 2 {
			return ""
		}
		token := normalizeBukpotTemplateToken(sub[1])
		value, ok := values[token]
		if !ok {
			missingTokens = append(missingTokens, sub[1])
			return ""
		}
		return value
	})

	filename := sanitizeBukpotFilename(rendered)
	warnings := make([]string, 0, 2)
	if filename == "" {
		fallback := sanitizeBukpotFilename(values["nomorbuktipotong"] + " - " + values["namapenerima"])
		if fallback == "" {
			fallback = sanitizeBukpotFilename(values["sourcename"])
		}
		if fallback == "" {
			fallback = bukpotDefaultFallbackFilename
		}
		filename = fallback
		warnings = append(warnings, "template resolved to empty filename, fallback name was used")
	}

	if len(missingTokens) > 0 {
		warnings = append(
			warnings,
			fmt.Sprintf("unknown template placeholder(s): %s", strings.Join(uniqueStrings(missingTokens), ", ")),
		)
	}

	if !strings.HasSuffix(strings.ToLower(filename), ".pdf") {
		filename += ".pdf"
	}
	return filename, uniqueStrings(warnings)
}

func normalizeBukpotTemplateToken(token string) string {
	normalized := strings.ToLower(strings.TrimSpace(token))
	normalized = strings.ReplaceAll(normalized, "_", "")
	return normalized
}

func sanitizeBukpotFilename(raw string) string {
	value := strings.TrimSpace(raw)
	value = strings.ReplaceAll(value, "/", "_")
	value = strings.ReplaceAll(value, "\\", "_")
	value = bukpotInvalidFilenameCharRegex.ReplaceAllString(value, " ")
	value = strings.ReplaceAll(value, "..", " ")
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, ".-_ ")
	return value
}

func sanitizeBukpotPathSegment(raw string) string {
	value := sanitizeBukpotFilename(raw)
	value = strings.ReplaceAll(value, "/", " ")
	value = strings.ReplaceAll(value, "\\", " ")
	return strings.TrimSpace(value)
}

func ensureUniqueBukpotFilename(name string, used map[string]struct{}) string {
	candidate := strings.TrimSpace(name)
	if candidate == "" {
		candidate = bukpotDefaultFallbackFilename + ".pdf"
	}

	ext := filepath.Ext(candidate)
	base := strings.TrimSuffix(candidate, ext)
	if ext == "" {
		ext = ".pdf"
	}

	lowered := strings.ToLower(candidate)
	if _, exists := used[lowered]; !exists {
		used[lowered] = struct{}{}
		return candidate
	}

	for i := 2; ; i++ {
		next := fmt.Sprintf("%s (%d)%s", base, i, ext)
		key := strings.ToLower(next)
		if _, exists := used[key]; exists {
			continue
		}
		used[key] = struct{}{}
		return next
	}
}

func buildBukpotZipArchive(entries []bukpotZipEntry, withCategoryFolder bool) ([]byte, error) {
	if len(entries) == 0 {
		return nil, errors.New("no renamed bukpot files to archive")
	}

	buf := bytes.NewBuffer(nil)
	zipWriter := zip.NewWriter(buf)

	for _, item := range entries {
		name := sanitizeBukpotFilename(item.Name)
		if name == "" {
			name = bukpotDefaultFallbackFilename + ".pdf"
		}
		if !strings.HasSuffix(strings.ToLower(name), ".pdf") {
			name += ".pdf"
		}

		entryPath := name
		if withCategoryFolder {
			category := sanitizeBukpotPathSegment(item.Category)
			if category == "" {
				category = bukpotUnknownCategory
			}
			entryPath = filepath.ToSlash(filepath.Join(category, name))
		}

		entry, err := zipWriter.Create(entryPath)
		if err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
		if _, err := entry.Write(item.Data); err != nil {
			_ = zipWriter.Close()
			return nil, err
		}
	}

	if err := zipWriter.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

func (p *PDFBukpotProcessor) attachBukpotZipOutput(
	ctx context.Context,
	collectionID string,
	zipBytes []byte,
	docType DocumentType,
	actionType string,
	result *ActionResult,
) error {
	docTypeSuffix := strings.TrimPrefix(string(docType), "pdf_")
	baseName := fmt.Sprintf("%s_%s", actionType, docTypeSuffix)
	if actionType == bukpotRenameActionType {
		baseName = fmt.Sprintf("rename_%s", docTypeSuffix)
	}
	zipName := fmt.Sprintf("%s_%s.zip", baseName, time.Now().Format("20060102_150405"))
	outputRef, err := p.fileStore.SaveArchive(ctx, collectionID, zipName, zipBytes)
	if err != nil {
		return err
	}

	zipSum := sha256.Sum256(zipBytes)
	result.Outputs = append(result.Outputs, ActionOutput{
		Kind:      "file",
		Name:      zipName,
		ObjectRef: outputRef,
		MimeType:  "application/zip",
		SizeBytes: int64(len(zipBytes)),
		Checksum:  hex.EncodeToString(zipSum[:]),
	})
	return nil
}
