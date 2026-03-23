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

type CollectionKind string

const (
	CollectionKindInvoiceCompany    CollectionKind = "invoice_company"
	CollectionKindTaxInvoiceCoretax CollectionKind = "tax_invoice_coretax"
	CollectionKindBukpotBPPU        CollectionKind = "bukpot_bppu"
	CollectionKindBukpotBP21        CollectionKind = "bukpot_bp21"
	CollectionKindBukpotBPA1        CollectionKind = "bukpot_bpa1"
	CollectionKindCashflowImport    CollectionKind = "cashflow_import"
)

func (k CollectionKind) IsValid() bool {
	switch k {
	case CollectionKindInvoiceCompany,
		CollectionKindTaxInvoiceCoretax,
		CollectionKindBukpotBPPU,
		CollectionKindBukpotBP21,
		CollectionKindBukpotBPA1,
		CollectionKindCashflowImport:
		return true
	default:
		return false
	}
}

type SourceFormat string

const (
	SourceFormatPDF  SourceFormat = "pdf"
	SourceFormatXLSX SourceFormat = "xlsx"
	SourceFormatCSV  SourceFormat = "csv"
)

func (f SourceFormat) IsValid() bool {
	switch f {
	case SourceFormatPDF, SourceFormatXLSX, SourceFormatCSV:
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

	NodeType       NodeType        `json:"nodeType"`
	CollectionKind *CollectionKind `json:"collectionKind,omitempty"`
	Phase          Phase           `json:"phase"`

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
	kind := CollectionKindInvoiceCompany
	return &Collection{
		ID:             id,
		UserID:         userID,
		Name:           name,
		NodeType:       NodeTypeCollection,
		CollectionKind: &kind,
		Phase:          PhaseReady,
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         StatusActive,
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
	collectionKind CollectionKind,
	now time.Time,
) *Collection {
	c := &Collection{
		ID:             id,
		UserID:         userID,
		Parent:         parent,
		Name:           name,
		NodeType:       NodeTypeCollection,
		CollectionKind: &collectionKind,
		Phase:          PhaseReady,
		CreatedAt:      now,
		UpdatedAt:      now,
		Status:         StatusActive,
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
