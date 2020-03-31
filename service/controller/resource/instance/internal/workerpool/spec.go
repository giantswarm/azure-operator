package workerpool

type Job interface {
	ID() string
	Run() error
	Finished() bool
}
