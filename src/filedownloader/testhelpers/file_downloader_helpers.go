package filedownloader_test

import (
	"crypto/md5"
	"fmt"
	"testing"

	"github.com/eriklarko/license-checker/src/filedownloader"
	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

type Thing struct {
	// implement filedownloader.Metadata interface
	Url string `yaml:"url"`
	Md5 string `yaml:"md5"`

	Name    string         `yaml:"-"`
	Path    string         `yaml:"-"`
	Content map[string]any `yaml:"-"`
}

func (t *Thing) GetUrl() string {
	return t.Url
}

func (t *Thing) GetMd5() string {
	return t.Md5
}

func (t *Thing) GetName() string {
	return t.Name
}

func (t *Thing) GetPath() string {
	return t.Path
}

func (t *Thing) SetPath(path string) {
	t.Path = path
}

func (t *Thing) SetUrl(url string) {
	t.Url = url
}

func (t *Thing) SetMd5(md5 string) {
	t.Md5 = md5
}

func (t *Thing) GetContent() map[string]any {
	return t.Content
}

func (t *Thing) SetContent(content map[string]any) {
	t.Content = content
}

type Thinger interface {
	GetName() string

	GetPath() string
	SetPath(path string)

	GetUrl() string
	SetUrl(url string)

	GetMd5() string
	SetMd5(md5 string)

	GetContent() map[string]any
	SetContent(content map[string]any)
}

func CreateFixtureWithThings[T Thinger](t *testing.T, things ...T) (*filedownloader.Service[T], *helpers_test.MockServer) {
	server := NewServerWithThings(t, things...)
	t.Cleanup(func() {
		server.Close()
	})

	sut := filedownloader.New[T]("things", server.URL()+"/metadata.yaml", t.TempDir())
	err := sut.DownloadMetadata()
	require.NoError(t, err)

	return sut, server
}

func NewServerWithThings[T Thinger](t *testing.T, things ...T) *helpers_test.MockServer {
	server := helpers_test.NewMockServer()

	metadata := make(map[string]any)
	for _, thing := range things {
		path := thing.GetPath()
		if path == "" {
			path = fmt.Sprintf("%s.yaml", thing.GetName())
		}

		thing.SetUrl(fmt.Sprintf("%s%s", server.URL(), path))

		if thing.GetMd5() == "" {
			// convert content to yaml and calculate md5
			if len(thing.GetContent()) == 0 {
				thing.SetContent(make(map[string]any))
			}
			contentBytes, err := yaml.Marshal(thing.GetContent())
			require.NoError(t, err)

			thing.SetMd5(fmt.Sprintf("%x", md5.Sum(contentBytes)))
		}

		metadata[thing.GetName()] = thing
		server.AddYamlResponse(thing.GetUrl(), thing.GetContent())
	}

	server.AddYamlResponse(server.URL()+"/metadata.yaml", metadata)

	return server
}
