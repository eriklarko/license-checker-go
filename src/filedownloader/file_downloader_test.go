package filedownloader_test

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/filedownloader"
	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDownloadMetadata(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	// set up a fake http server that will return the metadata
	httpMock := helpers_test.NewMockServer()
	defer httpMock.Close()

	cacheDir := t.TempDir()
	endpoint := httpMock.URL() + "/metadata.yaml"

	// define two things to download
	thing1Url := httpMock.URL() + "/thing1.yaml"
	thing1 := map[string]any{
		"foo": "bar",
	}
	httpMock.AddYamlResponse(thing1Url, thing1)

	thing2Url := httpMock.URL() + "/thing2.yaml"
	thing2 := map[string]any{
		"bar": "baz",
	}
	httpMock.AddYamlResponse(thing2Url, thing2)

	// and publish both curated things
	thingMetadata := map[string]any{
		"thing1": map[string]any{
			"md5": "73411061536ff8a32777eec043ece0e6",
			"url": thing1Url,
		},
		"thing2": map[string]any{
			"md5": "fad50251071a2532729e7f4beb79f8ca",
			"url": thing2Url,
		},
	}
	httpMock.AddYamlResponse(endpoint, thingMetadata)

	// download things
	sut := filedownloader.New("things", endpoint, cacheDir)
	err := sut.DownloadMetadata()
	require.NoError(t, err)

	// check that the lock file was downloaded, but nothing else
	assertFileExists(t,
		filepath.Join(cacheDir, "things-lock.yaml"),
		thingMetadata,
	)
	assert.NoFileExists(t, filepath.Join(cacheDir, "thing1.yaml"))
	assert.NoFileExists(t, filepath.Join(cacheDir, "thing2.yaml"))
}

type thing struct {
	name    string
	path    string
	md5     string
	content map[string]any
}

func TestDownload(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		someContent := map[string]any{
			"foo": "bar",
		}

		sut, _ := createFixtureWithThings(t,
			thing{
				name:    "thing",
				path:    "/thing.yaml",
				md5:     "721aad13918f292d25bc9dc7d61b0e9c",
				content: someContent,
			},
		)

		err := sut.Download("thing")
		require.NoError(t, err)

		// assert file was downloaded with correct contents
		assertFileExists(t,
			filepath.Join(sut.GetDownloadDir(), "thing.yaml"),
			someContent,
		)
	})

	t.Run("invalid md5", func(t *testing.T) {
		sut, httpMock := createFixtureWithThings(t,
			thing{
				name: "some-name",
				path: "/thing.yaml",
				md5:  "721aad13918f292d25bc9dc7d61b0e9c",
				content: map[string]any{
					"foo": "bar",
				},
			},
		)

		// overwrite what's being downloaded from /thing.yaml with something else
		httpMock.AddYamlResponse(httpMock.URL()+"/thing.yaml", map[string]any{
			"incorrect": "you are the weakest link",
		})

		err := sut.Download("some-name")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "md5 mismatch")
		assert.Contains(t, err.Error(), "some-name")
	})
}

func TestValidateDownloadedFiles(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, _ := createFixtureWithThings(t,
		thing{
			name: "some-name",
			path: "/thing.yaml",
			md5:  "721aad13918f292d25bc9dc7d61b0e9c",
			content: map[string]any{
				"foo": "bar",
			},
		},
	)

	err := sut.Download("some-name")
	require.NoError(t, err)

	t.Run("correct md5", func(t *testing.T) {
		// from the test set up above, we know that we've downloaded one list
		// successfully and because we've made no changes, it should all be
		// correct
		err := sut.TestValidateDownloadedFiles()
		require.NoError(t, err)
	})

	t.Run("incorrect md5", func(t *testing.T) {
		// overwrite the list file with different content
		err := os.WriteFile(filepath.Join(sut.GetDownloadDir(), "some-name.yaml"), []byte("You will be like us"), 0644)
		require.NoError(t, err)

		err = sut.TestValidateDownloadedFiles()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "md5 mismatch")
		assert.Contains(t, err.Error(), "some-name")
	})
}

func TestLockFileIsCached(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, _ := createFixtureWithThings(t,
		thing{
			name: "thing1",
			path: "thing1.yaml",
			md5:  "73411061536ff8a32777eec043ece0e6",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT"},
				"disallowed-licenses": []string{"GPL-3.0"},
			},
		},
		thing{
			name: "thing2",
			path: "thing2.yaml",
			md5:  "fad50251071a2532729e7f4beb79f8ca",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT", "Apache-2.0", "GPL-3.0"},
				"disallowed-licenses": []string{},
			},
		},
	)

	fmt.Println("FIXUTRE DONE")
	// get all things, and save the read things in `things1`
	things1, err := sut.GetLockFileContents()
	require.NoError(t, err)

	// change the lock file so that we can see a change in the returned things if
	// it is read again
	// get all again things, and save the read things in `things2`
	err = os.WriteFile(sut.GetLockFilePath(), []byte("Upgrade in progress"), 0644)
	require.NoError(t, err)

	things2, err := sut.GetLockFileContents()
	require.NoError(t, err)

	assert.Equal(t, things1, things2)
}

func createFixtureWithThings(t *testing.T, things ...thing) (*filedownloader.Service, *helpers_test.MockServer) {
	server := newServerWiththings(t, things...)
	t.Cleanup(func() {
		server.Close()
	})

	sut := filedownloader.New("things", server.URL()+"/metadata.yaml", t.TempDir())
	err := sut.DownloadMetadata()
	require.NoError(t, err)

	return sut, server
}

func newServerWiththings(t *testing.T, things ...thing) *helpers_test.MockServer {
	server := helpers_test.NewMockServer()

	metadata := make(map[string]any)
	for _, thing := range things {
		path := thing.path
		if path == "" {
			path = fmt.Sprintf("%s.yaml", thing.name)
		}

		url := fmt.Sprintf("%s%s", server.URL(), path)

		hash := thing.md5
		if hash == "" {
			// convert content to yaml and calculate md5
			if len(thing.content) == 0 {
				thing.content = make(map[string]any)
			}
			contentBytes, err := yaml.Marshal(thing.content)
			require.NoError(t, err)

			hash = fmt.Sprintf("%x", md5.Sum(contentBytes))
		}

		metadata[thing.name] = map[string]any{
			"url": url,
			"md5": hash,
		}

		server.AddYamlResponse(url, thing.content)
	}

	server.AddYamlResponse(server.URL()+"/metadata.yaml", metadata)

	return server
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
