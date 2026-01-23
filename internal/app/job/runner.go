package job

type JobRunner interface {
	Run(job Job) error
}
