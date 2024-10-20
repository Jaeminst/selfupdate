package selfupdate

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestReplaceURLTemplate(t *testing.T) {
	nochange := "http://localhost/nomad-windows-amd64.exe"
	change := "http://localhost/nomad-{{.OS}}-{{.Arch}}{{.Ext}}"
	expected := ""
	if runtime.GOOS == "windows" {
		expected = "http://localhost/nomad-" + runtime.GOOS + "-" + runtime.GOARCH + ".exe"
	} else {
		expected = "http://localhost/nomad-" + runtime.GOOS + "-" + runtime.GOARCH
	}

	r := replaceURLTemplate(nochange)
	assert.Equal(t, nochange, r)

	r = replaceURLTemplate(change)
	assert.NotEqual(t, change, r)
	assert.Equal(t, expected, r)
}
