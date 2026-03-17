package pdftool

type ExtractOptions struct {
	Layout   bool
	Table    bool
	NoPgBrk  bool
	Encoding string
}

func DefaultOptions() ExtractOptions {
	return ExtractOptions{
		Layout:   true,
		Table:    false,
		NoPgBrk:  true,
		Encoding: "UTF-8",
	}
}
