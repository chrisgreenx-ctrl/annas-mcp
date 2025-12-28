package modes

type SearchParams struct {
	SearchTerm string `json:"term" jsonschema:"Term to search for"`
}

type DownloadParams struct {
	BookHash string `json:"hash" jsonschema:"MD5 hash of the book to download"`
	Title    string `json:"title" jsonschema:"Book title, used for filename"`
	Format   string `json:"format" jsonschema:"Book format, for example pdf or epub"`
}
