package exercise

type File struct {
	Idx      *uint   `json:"idx,omitempty"`
	Language *string `json:"language"`
	Path     string  `json:"path"`
	Content  *string `json:"content,omitempty"`
}
