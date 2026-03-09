package daily

import "errors"

var (
	ErrNotExistingReport     = errors.New("no existing report found")
	ErrNotExistingRssContent = errors.New("no existing rss content found")
	ErrNoAudioURL            = errors.New("no audio url found")
	ErrStatusNotOK           = errors.New("unexpected status code")
	ErrBodyError             = errors.New("error body")
	ErrContentIsEmpty        = errors.New("content is empty")
	ErrContentIsToShort      = errors.New("content is to short")
)
