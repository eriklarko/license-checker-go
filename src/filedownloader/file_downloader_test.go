package filedownloader_test

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/filedownloader"
	filedownloader_test "github.com/eriklarko/license-checker/src/filedownloader/testhelpers"

	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	sut := filedownloader.New[*filedownloader_test.Thing]("things", endpoint, cacheDir)
	err := sut.DownloadMetadata()
	require.NoError(t, err)

	// check that the lock file was downloaded, but nothing else
	helpers_test.AssertYamlFileExists(t,
		filepath.Join(cacheDir, "things-lock.yaml"),
		thingMetadata,
	)
	assert.NoFileExists(t, filepath.Join(cacheDir, "thing1.yaml"))
	assert.NoFileExists(t, filepath.Join(cacheDir, "thing2.yaml"))
}

func TestDownload(t *testing.T) {
	t.Run("happy path", func(t *testing.T) {
		someContent := map[string]any{
			"foo": "bar",
		}

		sut, _ := filedownloader_test.CreateFixtureWithThings[*filedownloader_test.Thing](t,
			&filedownloader_test.Thing{
				Name:    "thing",
				Path:    "/thing.yaml",
				Md5:     "721aad13918f292d25bc9dc7d61b0e9c",
				Content: someContent,
			},
		)

		err := sut.Download("thing")
		require.NoError(t, err)

		// assert file was downloaded with correct contents
		helpers_test.AssertYamlFileExists(t,
			filepath.Join(sut.GetDownloadDir(), "thing.yaml"),
			someContent,
		)
	})

	t.Run("invalid md5", func(t *testing.T) {
		sut, httpMock := filedownloader_test.CreateFixtureWithThings(t,
			&filedownloader_test.Thing{
				Name: "some-name",
				Path: "/thing.yaml",
				Md5:  "721aad13918f292d25bc9dc7d61b0e9c",
				Content: map[string]any{
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

	sut, _ := filedownloader_test.CreateFixtureWithThings(t,
		&filedownloader_test.Thing{
			Name: "some-name",
			Path: "/thing.yaml",
			Md5:  "721aad13918f292d25bc9dc7d61b0e9c",
			Content: map[string]any{
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
		err := sut.ValidateDownloadedFiles()
		require.NoError(t, err)
	})

	t.Run("incorrect md5", func(t *testing.T) {
		// overwrite the list file with different content
		err := os.WriteFile(filepath.Join(sut.GetDownloadDir(), "some-name.yaml"), []byte("You will be like us"), 0644)
		require.NoError(t, err)

		err = sut.ValidateDownloadedFiles()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "md5 mismatch")
		assert.Contains(t, err.Error(), "some-name")
	})
}

func TestLockFileIsCached(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, _ := filedownloader_test.CreateFixtureWithThings(t,
		&filedownloader_test.Thing{
			Name: "thing1",
			Path: "thing1.yaml",
			Md5:  "73411061536ff8a32777eec043ece0e6",
			Content: map[string]any{
				"foo:": "bar",
			},
		},
		&filedownloader_test.Thing{
			Name: "thing2",
			Path: "thing2.yaml",
			Md5:  "fad50251071a2532729e7f4beb79f8ca",
			Content: map[string]any{
				"bar:": "baz",
			},
		},
	)

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
