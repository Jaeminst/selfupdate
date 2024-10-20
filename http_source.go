package selfupdate

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"text/template"
)

// HTTPSource provide a Source that will download the update from a HTTP url.
// It is expecting the signature file to be served at ${URL}.ed25519
type HTTPSource struct {
	client  *http.Client
	baseURL string
}

var _ Source = (*HTTPSource)(nil)

type platform struct {
	OS         string
	Arch       string
	Ext        string
	Executable string
}

// NewHTTPSource provide a selfupdate.Source that will fetch the specified base URL
// for update and signature using the http.Client provided. To help into providing
// cross platform application, the base is actually a Go Template string where the
// following parameter are recognized:
// {{.OS}} will be filled by the runtime OS name
// {{.Arch}} will be filled by the runtime Arch name
// {{.Ext}} will be filled by the executable expected extension for the OS
// As an example the following string `http://localhost/myapp-{{.OS}}-{{.Arch}}{{.Ext}}`
// would fetch on Windows AMD64 the following URL: `http://localhost/myapp-windows-amd64.exe`
// and on Linux AMD64: `http://localhost/myapp-linux-amd64`.
func NewHTTPSource(client *http.Client, base string) Source {
	if client == nil {
		client = http.DefaultClient
	}

	base = replaceURLTemplate(base)

	return &HTTPSource{client: client, baseURL: base}
}

// Get will return if it succeed an io.ReaderCloser to the new executable being downloaded and its length
func (h *HTTPSource) Get() (io.ReadCloser, int64, error) {
	request, err := http.NewRequest("GET", h.baseURL, nil)
	if err != nil {
		return nil, 0, err
	}

	response, err := h.client.Do(request)
	if err != nil {
		return nil, 0, err
	}

	if response.StatusCode != 200 {
		return nil, 0, fmt.Errorf("failed to get content")
	}

	return response.Body, response.ContentLength, nil
}

func replaceURLTemplate(base string) string {
	ext := ""
	if runtime.GOOS == "windows" {
		ext = ".exe"
	}

	p := platform{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
		Ext:  ext,
	}

	exe, err := ExecutableRealPath()
	if err != nil {
		exe = filepath.Base(os.Args[0])
	} else {
		exe = filepath.Base(exe)
	}
	if runtime.GOOS == "windows" {
		p.Executable = exe[:len(exe)-len(".exe")]
	} else {
		p.Executable = exe
	}

	t, err := template.New("platform").Parse(base)
	if err != nil {
		return base
	}

	buf := &strings.Builder{}
	err = t.Execute(buf, p)
	if err != nil {
		return base
	}
	return buf.String()
}
