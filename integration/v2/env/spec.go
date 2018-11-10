package env

type TestedVersion string

func (t TestedVersion) String() string {
	return string(t)
}

const (
	TestedVersionLast TestedVersion = "last"
	TestedVersionPrev TestedVersion = "prev"
)
