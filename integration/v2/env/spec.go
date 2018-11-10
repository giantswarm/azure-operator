package env

type TestedVersion string

func (t TestedVersion) String() {
	return string(t)
}

const (
	TestedVersionLast TestedVersion = "last"
	TestedVersionPrev TestedVersion = "prev"
)
