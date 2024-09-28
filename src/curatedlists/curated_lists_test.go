package curatedlists_test

import (
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlists"
	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDownloadLists(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// set up a fake http server that will return our curated lists
	httpMock := helpers_test.NewMockServer()
	defer httpMock.Close()

	conf := &config.Config{
		CacheDir:           helpers_test.CreateTempDir(t, "license-checker-test-cache-*"),
		CuratedlistsSource: httpMock.URL() + "/curated-lists.yaml",
	}

	// define two curated lists
	list1Url := httpMock.URL() + "/list1.yaml"
	list1 := map[string]any{
		"allowed-licenses":    []string{"MIT"},
		"disallowed-licenses": []string{"GPL-3.0"},
	}
	httpMock.AddYamlResponse(list1Url, list1)

	list2Url := httpMock.URL() + "/list2.yaml"
	list2 := map[string]any{
		"allowed-licenses":    []string{"MIT", "Apache-2.0", "GPL-3.0"},
		"disallowed-licenses": []string{},
	}
	httpMock.AddYamlResponse(list2Url, list2)

	// and publish both curated lists
	curatedListsContent := map[string]any{
		"list1": map[string]any{
			"md5":         "73411061536ff8a32777eec043ece0e6",
			"url":         list1Url,
			"description": "A silly list that is incredibly conservative",
		},
		"list2": map[string]any{
			"md5":         "fad50251071a2532729e7f4beb79f8ca",
			"url":         list2Url,
			"description": "A more permissive list",
		},
	}
	httpMock.AddYamlResponse(conf.CuratedlistsSource, curatedListsContent)

	// download lists
	sut := curatedlists.New(conf, http.DefaultTransport)
	err := sut.DownloadCuratedLists()
	require.NoError(t, err)

	// check that the lists were downloaded
	assertFileExists(t,
		filepath.Join(conf.CacheDir, "list-lock.yaml"),
		curatedListsContent,
	)
	assertFileExists(t,
		filepath.Join(conf.CacheDir, "list1.yaml"),
		list1,
	)
	assertFileExists(t,
		filepath.Join(conf.CacheDir, "list2.yaml"),
		list2,
	)
}

func assertFileExists(t *testing.T, path string, content map[string]any) {
	t.Helper()

	assert.FileExists(t, path)

	// check that the file contains the expected content
	fileContent, err := os.ReadFile(path)
	require.NoError(t, err)

	contentBytes, err := yaml.Marshal(content)
	require.NoError(t, err)

	assert.YAMLEq(t, string(contentBytes), string(fileContent))
}
