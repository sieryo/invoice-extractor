package file

type StorageUsage struct {
	TempBytes  int64 `json:"temp_bytes"`
	FilesBytes int64 `json:"files_bytes"`
	AuditBytes int64 `json:"audit_bytes"`
	TotalBytes int64 `json:"total_bytes"`
}
