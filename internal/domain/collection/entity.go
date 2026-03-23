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
	DocumentTypePDFBukpotBPPU DocumentType = "pdf_bppu"
	DocumentTypePDFBukpotBP21 DocumentType = "pdf_bp21"
	DocumentTypePDFBukpotBPA1 DocumentType = "pdf_bpa1"
)

func (d DocumentType) IsValid() bool {
	switch d {
	case DocumentTypePDFInvoice,
		DocumentTypePDFTaxInvoice,
		DocumentTypePDFBukpotBPPU,
		DocumentTypePDFBukpotBP21,
		DocumentTypePDFBukpotBPA1:
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

	UserID string  `json:"userId"`
	Name   string  `json:"name"`
	Status Status  `json:"status,omitempty"` // legacy field for old FE contracts
	Parent *string `json:"parentId,omitempty"`

	NodeType     NodeType      `json:"nodeType"`
	DocumentType *DocumentType `json:"documentType,omitempty"`
	Phase        Phase         `json:"phase"`

	TotalCount     int `json:"totalCount"`
	ReadyCount     int `json:"readyCount"`
	WarningCount   int `json:"warningCount"`
	FailedCount    int `json:"failedCount"`
	DuplicateCount int `json:"duplicateCount"`

	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`

	FrozenAt *time.Time `json:"frozenAt,omitempty"`
	FrozenBy *string    `json:"frozenBy,omitempty"`

	DeletedAt     *time.Time `json:"deletedAt,omitempty"`
	DeletedBy     *string    `json:"deletedBy,omitempty"`
	DeleteReason  *string    `json:"deleteReason,omitempty"`
	LegacyExpired *time.Time `json:"expiredAt,omitempty"` // legacy compatibility
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

func (c *Collection) IsFrozen() bool {
	return c.FrozenAt != nil
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
	case c.IsFrozen():
		c.Status = "frozen"
	case c.IsFolder():
		c.Status = "folder"
	default:
		c.Status = StatusActive
	}
}
