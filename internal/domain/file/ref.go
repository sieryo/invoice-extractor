package file

type FileRef struct {
	ID           string `json:"id"`
	CollectionID string `json:"collection_id"`
	Name         string `json:"name"`
}

type ResolvedFile struct {
	FileRef
	Path string `json:"path"`
}
