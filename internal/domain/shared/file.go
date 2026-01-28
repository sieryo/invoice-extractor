package shared

type FileResultError struct {
	FileID   string
	FileName string
	Err      error
}
