package configlayout

type SectionSpec struct {
	Key         string   `json:"key"`
	Title       string   `json:"title"`
	Description string   `json:"description,omitempty"`
	Columns     int      `json:"columns,omitempty"`
	FieldKeys   []string `json:"fieldKeys,omitempty"`
}
