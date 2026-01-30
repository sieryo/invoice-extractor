package file

type FileState string

const (
	FileStateTemp  FileState = "temp"
	FileStateFinal FileState = "final"
)

type FileObject struct {
	ID           string    `json:"id"`
	CollectionID string    `json:"collection_id"`
	Name         string    `json:"name"`
	Path         string    `json:"path"`
	State        FileState `json:"state"`
	Size         int64     `json:"size"`
	MimeType     string    `json:"mime_type"`
}

func NewTempFile(
	id string,
	collectionID string,
	name string,
	path string,
	size int64,
	mimeType string,
) FileObject {
	return FileObject{
		ID:           id,
		CollectionID: collectionID,
		Name:         name,
		Path:         path,
		State:        FileStateTemp,
		Size:         size,
		MimeType:     mimeType,
	}
}

func (f FileObject) Commit(newPath string) (FileObject, error) {
	if f.State != FileStateTemp {
		return FileObject{}, ErrInvalidFileStateTransition
	}

	f.Path = newPath
	f.State = FileStateFinal
	return f, nil
}

func (f FileObject) IsTemp() bool {
	return f.State == FileStateTemp
}

func (f FileObject) IsFinal() bool {
	return f.State == FileStateFinal
}

type UploadInput struct {
	Name string
	Data []byte
}
