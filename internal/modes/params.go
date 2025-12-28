package modes

type SearchParams struct {
	SearchTerm string `json:"term" jsonschema:"description=Term to search for"`
}

type DownloadParams struct {
	BookHash string `json:"hash" jsonschema:"description=MD5 hash of the book to download"`
	Title    string `json:"title" jsonschema:"description=Book title, used for filename"`
	Format   string `json:"format" jsonschema:"description=Book format, for example pdf or epub"`
}
