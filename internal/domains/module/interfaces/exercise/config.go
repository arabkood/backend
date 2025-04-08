package exercise

type ExerciseFiles struct {
	Solution []File  `json:"solution"`
	Test     *[]File `json:"test,omitempty"`
	Example  *[]File `json:"example,omitempty"`
}

type ExerciseConfig struct {
	Files  ExerciseFiles          `json:"files"`
	Image  string                 `json:"image"`
	Blurb  *string                `json:"blurb,omitempty"`
	Custom map[string]interface{} `json:"custom,omitempty"`
}
