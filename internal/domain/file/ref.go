package file

type FileRef struct {
	ID           string
	CollectionID string
	Name         string
}

type ResolvedFile struct {
	FileRef
	Path string
}
