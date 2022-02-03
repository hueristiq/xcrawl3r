package runner

type Results struct {
	URLs    []string `json:"urls,omitempty"`
	Buckets []string `json:"s3,omitempty"`
}
