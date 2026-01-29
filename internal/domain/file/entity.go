package file

type FileState string

const (
	FileStateTemp  FileState = "temp"
	FileStateFinal FileState = "final"
)

type FileObject struct {
	ID           string
	CollectionID string
	Name         string
	Path         string
	State        FileState
}

// NewTempFile adalah cara resmi bikin file temp
func NewTempFile(
	id string,
	collectionID string,
	name string,
	path string,
) FileObject {
	return FileObject{
		ID:           id,
		CollectionID: collectionID,
		Name:         name,
		Path:         path,
		State:        FileStateTemp,
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
