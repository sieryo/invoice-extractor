package collection

import "time"

// Legacy status type kept for compatibility with old callers.
type Status string

const (
	StatusActive    Status = "active"
	StatusCommitted Status = "committed"
	StatusExpired   Status = "expired"
)

type NodeType string

const (
	NodeTypeFolder     NodeType = "folder"
	NodeTypeCollection NodeType = "collection"
)

type DocumentType string

const (
	DocumentTypePDFInvoice    DocumentType = "pdf_invoice"
	DocumentTypePDFTaxInvoice DocumentType = "pdf_tax_invoice"
	DocumentTypeXLSXCashflow  DocumentType = "xlsx_cashflow"
)

func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentTypePDFInvoice, DocumentTypePDFTaxInvoice, DocumentTypeXLSXCashflow:
		return true
	default:
		return false
	}
}

type Phase string

const (
	PhaseReady      Phase = "ready"
	PhaseUploading  Phase = "uploading"
	PhaseProcessing Phase = "processing"
)

type Collection struct {
	ID string `json:"id"`

	UserID string  `json:"user_id"`
	Name   string  `json:"name"`
	Status Status  `json:"status,omitempty"` // legacy field for old FE contracts
	Parent *string `json:"parent_id,omitempty"`

	NodeType     NodeType      `json:"node_type"`
	DocumentType *DocumentType `json:"document_type,omitempty"`
	Phase        Phase         `json:"phase"`

	TotalCount     int `json:"total_count"`
	ReadyCount     int `json:"ready_count"`
	WarningCount   int `json:"warning_count"`
	FailedCount    int `json:"failed_count"`
	DuplicateCount int `json:"duplicate_count"`

	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`

	DeletedAt     *time.Time `json:"deleted_at,omitempty"`
	DeletedBy     *string    `json:"deleted_by,omitempty"`
	DeleteReason  *string    `json:"delete_reason,omitempty"`
	LegacyExpired *time.Time `json:"expired_at,omitempty"` // legacy compatibility
}

func NewCollection(id string, userID string, name string, now time.Time) *Collection {
	docType := DocumentTypePDFInvoice
	return &Collection{
		ID:           id,
		UserID:       userID,
		Name:         name,
		NodeType:     NodeTypeCollection,
		DocumentType: &docType,
		Phase:        PhaseReady,
		CreatedAt:    now,
		UpdatedAt:    now,
		Status:       StatusActive,
	}
}

func NewFolder(id string, userID string, parent *string, name string, now time.Time) *Collection {
	return &Collection{
		ID:        id,
		UserID:    userID,
		Parent:    parent,
		Name:      name,
		NodeType:  NodeTypeFolder,
		Phase:     PhaseReady,
		CreatedAt: now,
		UpdatedAt: now,
		Status:    StatusActive,
	}
}

func NewTypedCollection(
	id string,
	userID string,
	parent *string,
	name string,
	documentType DocumentType,
	now time.Time,
) *Collection {
	c := &Collection{
		ID:           id,
		UserID:       userID,
		Parent:       parent,
		Name:         name,
		NodeType:     NodeTypeCollection,
		DocumentType: &documentType,
		Phase:        PhaseReady,
		CreatedAt:    now,
		UpdatedAt:    now,
		Status:       StatusActive,
	}
	c.SyncLegacyStatus()
	return c
}

func (c *Collection) IsFolder() bool {
	return c.NodeType == NodeTypeFolder
}

func (c *Collection) IsCollection() bool {
	return c.NodeType == NodeTypeCollection
}

func (c *Collection) IsActive() bool {
	return c.IsCollection() && c.DeletedAt == nil
}

func (c *Collection) IsCommitted() bool {
	return false
}

func (c *Collection) IsExpired() bool {
	return false
}

func (c *Collection) SetPhase(phase Phase) {
	c.Phase = phase
}

func (c *Collection) SyncLegacyStatus() {
	switch {
	case c.DeletedAt != nil:
		c.Status = "deleted"
	case c.IsFolder():
		c.Status = "folder"
	default:
		c.Status = StatusActive
	}
}
