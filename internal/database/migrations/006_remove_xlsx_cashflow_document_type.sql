PRAGMA foreign_keys = ON;

DELETE FROM collection_action_outputs
WHERE action_id IN (
	SELECT id FROM collection_actions WHERE document_type = 'xlsx_cashflow'
);

DELETE FROM collection_action_items
WHERE action_id IN (
	SELECT id FROM collection_actions WHERE document_type = 'xlsx_cashflow'
);

DELETE FROM collection_actions
WHERE document_type = 'xlsx_cashflow';

DELETE FROM collection_history_items
WHERE document_type = 'xlsx_cashflow';

DELETE FROM upload_session_chunks
WHERE session_id IN (
	SELECT id FROM upload_sessions WHERE document_type = 'xlsx_cashflow'
);

DELETE FROM upload_sessions
WHERE document_type = 'xlsx_cashflow';

DELETE FROM documents
WHERE document_type = 'xlsx_cashflow';

DELETE FROM collections
WHERE node_type = 'collection'
  AND document_type = 'xlsx_cashflow';

DROP TRIGGER IF EXISTS trg_no_xlsx_collections_insert;
CREATE TRIGGER trg_no_xlsx_collections_insert
BEFORE INSERT ON collections
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_collections_update;
CREATE TRIGGER trg_no_xlsx_collections_update
BEFORE UPDATE OF document_type ON collections
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_documents_insert;
CREATE TRIGGER trg_no_xlsx_documents_insert
BEFORE INSERT ON documents
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_documents_update;
CREATE TRIGGER trg_no_xlsx_documents_update
BEFORE UPDATE OF document_type ON documents
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_upload_sessions_insert;
CREATE TRIGGER trg_no_xlsx_upload_sessions_insert
BEFORE INSERT ON upload_sessions
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_upload_sessions_update;
CREATE TRIGGER trg_no_xlsx_upload_sessions_update
BEFORE UPDATE OF document_type ON upload_sessions
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_collection_history_items_insert;
CREATE TRIGGER trg_no_xlsx_collection_history_items_insert
BEFORE INSERT ON collection_history_items
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_collection_history_items_update;
CREATE TRIGGER trg_no_xlsx_collection_history_items_update
BEFORE UPDATE OF document_type ON collection_history_items
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_collection_actions_insert;
CREATE TRIGGER trg_no_xlsx_collection_actions_insert
BEFORE INSERT ON collection_actions
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;

DROP TRIGGER IF EXISTS trg_no_xlsx_collection_actions_update;
CREATE TRIGGER trg_no_xlsx_collection_actions_update
BEFORE UPDATE OF document_type ON collection_actions
FOR EACH ROW
WHEN NEW.document_type = 'xlsx_cashflow'
BEGIN
	SELECT RAISE(ABORT, 'xlsx_cashflow document type is no longer supported');
END;
