package crawler

import (
	"context"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/yugjain1212/crawliq/internal/workers"
)

type Crawler struct {
	fetcher *Fetcher
}

func NewCrawler(fetcher *Fetcher) *Crawler {
	return &Crawler{
		fetcher: fetcher,
	}
}
func (cr *Crawler) ProcessJob(job workers.Job, timeout time.Duration) workers.Result {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	fetchResult, err := cr.fetcher.Fetch(ctx, job.URL)
	if err != nil {
		errMsg := err.Error()
		log.Debug().Str("url", job.URL).Err(err).Msg("fetch failed")
		return workers.Result{
			Job:   job,
			Error: &errMsg,
		}
	}
	result := workers.Result{
		Job:            job,
		StatusCode:     intPtr(fetchResult.StatusCode),
		ContentType:    strPtr(fetchResult.ContentType),
		ContentLength:  int64Ptr(fetchResult.ContentLength),
		ResponseTimeMs: intPtr(int(fetchResult.ResponseTime.Milliseconds())),
	}

	if IsHTML(fetchResult.ContentType) {
		parsed, err := ParseHTML(fetchResult.Body)
		if err != nil {
			// A parse failure doesn't erase the fetch data we already
			// have (status code, timing, etc.) — we still return those,
			// we just have no title to report.
			log.Debug().Str("url", job.URL).Err(err).Msg("html parse failed")
		} else {
			result.Title = strPtr(parsed.Title)
		}
		htmlStr := string(fetchResult.Body)
		result.HTML = &htmlStr
	}

	return result
}

func intPtr(v int) *int       { return &v }
func strPtr(v string) *string { return &v }
func int64Ptr(v int64) *int64 { return &v }
