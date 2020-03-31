package workerpool

type Job interface {
	Run() error
	Finished() bool
}
