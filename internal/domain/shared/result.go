package shared

type FileResultError struct {
	FileID   string `json:"file_id"`
	FileName string `json:"file_name"`
	Error    string `json:"error"`
}
