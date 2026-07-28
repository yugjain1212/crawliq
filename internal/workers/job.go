package workers

import "time"

type Job struct {
	CRAWLID int64
	URL     string
}

type Result struct {
	Job            Job
	StatusCode     *int
	ContentType    *string
	ContentLength  *string
	ResponseTimeMs *int
	Title          *string
	HTML           *string
	Error          *string
}

func (r *Result) succedded() bool {
	return r.Error == nil && r.StatusCode != nil
}

type ProcessFunc func(job Job, timeout time.Duration) Result
