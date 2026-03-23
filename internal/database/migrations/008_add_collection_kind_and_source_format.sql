ALTER TABLE collections ADD COLUMN collection_kind TEXT;

UPDATE collections
SET collection_kind = CASE document_type
	WHEN 'pdf_invoice' THEN 'invoice_company'
	WHEN 'pdf_tax_invoice' THEN 'tax_invoice_coretax'
	WHEN 'pdf_bppu' THEN 'bukpot_bppu'
	WHEN 'pdf_bp21' THEN 'bukpot_bp21'
	WHEN 'pdf_bpa1' THEN 'bukpot_bpa1'
	ELSE NULL
END
WHERE node_type = 'collection'
  AND collection_kind IS NULL;

ALTER TABLE documents ADD COLUMN collection_kind TEXT;
ALTER TABLE documents ADD COLUMN source_format TEXT;

UPDATE documents
SET
	collection_kind = CASE document_type
		WHEN 'pdf_invoice' THEN 'invoice_company'
		WHEN 'pdf_tax_invoice' THEN 'tax_invoice_coretax'
		WHEN 'pdf_bppu' THEN 'bukpot_bppu'
		WHEN 'pdf_bp21' THEN 'bukpot_bp21'
		WHEN 'pdf_bpa1' THEN 'bukpot_bpa1'
		ELSE NULL
	END,
	source_format = COALESCE(source_format, 'pdf')
WHERE collection_kind IS NULL
   OR source_format IS NULL;

ALTER TABLE upload_sessions ADD COLUMN collection_kind TEXT;
ALTER TABLE upload_sessions ADD COLUMN source_format TEXT;

UPDATE upload_sessions
SET
	collection_kind = CASE document_type
		WHEN 'pdf_invoice' THEN 'invoice_company'
		WHEN 'pdf_tax_invoice' THEN 'tax_invoice_coretax'
		WHEN 'pdf_bppu' THEN 'bukpot_bppu'
		WHEN 'pdf_bp21' THEN 'bukpot_bp21'
		WHEN 'pdf_bpa1' THEN 'bukpot_bpa1'
		ELSE NULL
	END,
	source_format = COALESCE(source_format, 'pdf')
WHERE collection_kind IS NULL
   OR source_format IS NULL;

ALTER TABLE collection_history_items ADD COLUMN collection_kind TEXT;
ALTER TABLE collection_history_items ADD COLUMN source_format TEXT;

UPDATE collection_history_items
SET
	collection_kind = CASE document_type
		WHEN 'pdf_invoice' THEN 'invoice_company'
		WHEN 'pdf_tax_invoice' THEN 'tax_invoice_coretax'
		WHEN 'pdf_bppu' THEN 'bukpot_bppu'
		WHEN 'pdf_bp21' THEN 'bukpot_bp21'
		WHEN 'pdf_bpa1' THEN 'bukpot_bpa1'
		ELSE NULL
	END,
	source_format = COALESCE(source_format, 'pdf')
WHERE collection_kind IS NULL
   OR source_format IS NULL;

ALTER TABLE collection_actions ADD COLUMN collection_kind TEXT;
ALTER TABLE collection_actions ADD COLUMN source_format TEXT;

UPDATE collection_actions
SET
	collection_kind = CASE document_type
		WHEN 'pdf_invoice' THEN 'invoice_company'
		WHEN 'pdf_tax_invoice' THEN 'tax_invoice_coretax'
		WHEN 'pdf_bppu' THEN 'bukpot_bppu'
		WHEN 'pdf_bp21' THEN 'bukpot_bp21'
		WHEN 'pdf_bpa1' THEN 'bukpot_bpa1'
		ELSE NULL
	END,
	source_format = COALESCE(source_format, 'pdf')
WHERE collection_kind IS NULL
   OR source_format IS NULL;

CREATE INDEX IF NOT EXISTS idx_collections_collection_kind
ON collections (collection_kind, deleted_at);

CREATE INDEX IF NOT EXISTS idx_documents_kind_format_active
ON documents (collection_id, collection_kind, source_format, deleted_at);

CREATE INDEX IF NOT EXISTS idx_upload_sessions_kind_format
ON upload_sessions (collection_id, collection_kind, source_format, status);

CREATE INDEX IF NOT EXISTS idx_collection_actions_kind_format
ON collection_actions (collection_id, collection_kind, source_format, created_at DESC);
