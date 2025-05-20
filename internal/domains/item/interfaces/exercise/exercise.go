package exercise

// type Exercise struct {
// 	Image        string `json:"image" db:"image"`
// 	Instructions string `json:"docs" db:"docs"`
// 	Hints        string `json:"hints" db:"hints"`
// 	Files        []File `json:"files" db:"files"`
// 	TestFiles    []File `json:"test_files" db:"test_files"`
// }

type CodeConfig struct {
	Image string            `json:"image"`
	Files []CodeFileConfig  `json:"files"`
	Flags map[string]string `json:"flags"`
}

type CodeFileConfig struct {
	Idx  int    `json:"idx"`
	Path string `json:"path"`
	Lang string `json:"lang"`
	Ro   *bool  `json:"ro,omitempty"` // Optional boolean
}

type Code struct {
	Config  CodeConfig        `json:"config"`
	Files   map[string]string `json:"files"`
	Docs    map[string]string `json:"docs"`
	Example map[string]string `json:"example,omitempty"` // Optional
}

type TestResult struct {
	Name     string `json:"name"`
	Status   string `json:"status"` // 'pass' | 'fail'
	Message  string `json:"message,omitempty"`
	TestCode string `json:"test_code,omitempty"`
}

type CodeResults struct {
	Status   string `json:"status"` // 'wait' | string
	XpReward int    `json:"xp_reward"`
	Results  struct {
		Status  string       `json:"status"` // 'error' | 'pass' | 'fail'
		Tests   []TestResult `json:"tests"`
		Message string       `json:"message,omitempty"`
	} `json:"results"`
}
