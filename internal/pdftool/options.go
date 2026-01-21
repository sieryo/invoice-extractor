package pdftool

type ExtractOptions struct {
	Layout   bool
	NoPgBrk  bool
	Encoding string
}

func DefaultOptions() ExtractOptions {
	return ExtractOptions{
		Layout:   true,
		NoPgBrk:  true,
		Encoding: "UTF-8",
	}
}
