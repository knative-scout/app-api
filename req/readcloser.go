package req

import (
	"io"
)

// ReaderDummyCloser implements the Close method ontop of an io.Reader
// Used to pass an io.Reader for net/http.Request.Body
type ReaderDummyCloser struct {
	io.Reader
}

// Close implements a meaningless close method for ReaderDummyCloser
func (b ReaderDummyCloser) Close() error {
	return nil
}
