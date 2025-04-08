package exercise

type Exercise struct {
	Image        string `json:"image" db:"image"`
	Instructions string `json:"docs" db:"docs"`
	Hints        string `json:"hints" db:"hints"`
	Files        []File `json:"files" db:"files"`
	TestFiles    []File `json:"test_files" db:"test_files"`
}
