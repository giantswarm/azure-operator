package workerpool

type Job interface {
	ID() string
	Finished() bool
	Run() error
}
