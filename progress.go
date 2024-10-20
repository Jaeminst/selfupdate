package selfupdate

import (
	"io"

	"github.com/schollz/progressbar/v3"
)

// ProgressReader is a wrapper around io.Reader to add a progress bar
type progressReader struct {
	io.Reader
	bar *progressbar.ProgressBar
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.Reader.Read(p)
	pr.bar.Add(n)
	return n, err
}
