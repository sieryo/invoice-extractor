package extract

type Payload struct {
	PDFPaths []string `json:"pdf_paths"`
	Template *string  `json:"template,omitempty"`
}
