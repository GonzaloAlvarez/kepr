package store

const (
	TypeDir      = "dir"
	TypePassword = "password"
	TypeFile     = "file"
)

type Metadata struct {
	Path string `json:"path"`
	Type string `json:"type"`
}
