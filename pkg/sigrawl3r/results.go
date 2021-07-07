package sigrawl3r

type Results struct {
	URLs    []string `json:"urls,omitempty"`
	Buckets []string `json:"s3,omitempty"`
}
