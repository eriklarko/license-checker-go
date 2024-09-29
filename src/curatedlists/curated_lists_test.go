package curatedlists_test

import (
	"crypto/md5"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlists"
	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/samber/lo"
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

	t.Run("happy path", func(t *testing.T) {
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
	})

	t.Run("invalid md5", func(t *testing.T) {
		// define a curated list
		httpMock.Reset()
		listUrl := httpMock.URL() + "/list.yaml"
		list := map[string]any{
			"allowed-licenses":    []string{"MIT"},
			"disallowed-licenses": []string{"GPL-3.0"},
		}
		httpMock.AddYamlResponse(listUrl, list)

		// and publish it, with incorrect md5
		curatedListsContent := map[string]any{
			"list": map[string]any{
				"md5": "Upgrade in progress",
				"url": listUrl,
			},
		}
		httpMock.AddYamlResponse(conf.CuratedlistsSource, curatedListsContent)

		// download lists
		sut := curatedlists.New(conf, http.DefaultTransport)
		err := sut.DownloadCuratedLists()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "md5")
	})
}

func TestGetAllLists(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, httpMock, _ := createFixtureWithDownloadedLists(t,
		List{
			name: "list1",
			path: "list1.yaml",
			md5:  "73411061536ff8a32777eec043ece0e6",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT"},
				"disallowed-licenses": []string{"GPL-3.0"},
			},
		},
		List{
			name: "list2",
			path: "list2.yaml",
			md5:  "fad50251071a2532729e7f4beb79f8ca",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT", "Apache-2.0", "GPL-3.0"},
				"disallowed-licenses": []string{},
			},
		},
	)

	// get all lists
	lists, err := sut.GetAllLists()
	require.NoError(t, err)

	// check that the lists were downloaded
	assert.Equal(t, curatedlists.ListMetadata{
		"list1": curatedlists.ListInfo{
			Md5:         "73411061536ff8a32777eec043ece0e6",
			Url:         httpMock.URL() + "/list1.yaml",
			Description: "",
		},
		"list2": curatedlists.ListInfo{
			Md5:         "fad50251071a2532729e7f4beb79f8ca",
			Url:         httpMock.URL() + "/list2.yaml",
			Description: "",
		},
	}, lists)
}

func TestGetHighestRatedList(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	testCases := map[string]struct {
		lists            []List
		expectedListName string
	}{
		"empty list": {
			lists:            nil,
			expectedListName: "",
		},
		"single list": {
			lists: []List{
				{
					name:        "list1",
					description: "list1 description",
					rating:      0.5,
				},
			},
			expectedListName: "list1",
		},
		"multiple lists": {
			lists: []List{
				{
					name:        "list1",
					description: "list1 description",
					rating:      0.5,
				},
				{
					name:        "list2",
					description: "list2 description",
					rating:      0.9,
				},
				{
					name:        "list3",
					description: "list3 description",
					rating:      0.7,
				},
			},
			expectedListName: "list2",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			sut, _, _ := createFixtureWithDownloadedLists(t, tc.lists...)

			var expectedList List
			if len(tc.lists) == 0 {
				// if there are no lists, the we expect "" for name and description
				expectedList = List{
					name:        "",
					description: "",
				}
			} else {

				// to avoid having to specify both list name and description in
				// the test case, we infer the expected description here by
				// searching for the list with the expected name
				var found bool
				expectedList, found = lo.Find(tc.lists, func(list List) bool {
					return list.name == tc.expectedListName
				})
				require.True(t, found, "expected list not found in test case")
			}

			listName, listDescription, err := sut.GetHighlyRatedList()
			require.NoError(t, err)
			assert.Equal(t, expectedList.name, listName)
			assert.Equal(t, expectedList.description, listDescription)
		})
	}
}

func TestLockFileIsCached(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, _, _ := createFixtureWithDownloadedLists(t,
		List{
			name: "list1",
			path: "list1.yaml",
			md5:  "73411061536ff8a32777eec043ece0e6",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT"},
				"disallowed-licenses": []string{"GPL-3.0"},
			},
		},
		List{
			name: "list2",
			path: "list2.yaml",
			md5:  "fad50251071a2532729e7f4beb79f8ca",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT", "Apache-2.0", "GPL-3.0"},
				"disallowed-licenses": []string{},
			},
		},
	)

	fmt.Println("FIXUTRE DONE")
	// get all lists, and save the read lists in `lists1`
	lists1, err := sut.GetAllLists()
	require.NoError(t, err)

	// change the lock file so that we can see a change in the returned lists if
	// it is read again
	// get all again lists, and save the read lists in `lists2`
	err = os.WriteFile(sut.GetLockFilePath(), []byte("Upgrade in progress"), 0644)
	require.NoError(t, err)

	lists2, err := sut.GetAllLists()
	require.NoError(t, err)

	assert.Equal(t, lists1, lists2)
}

func TestValidateLocalLists(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, _, conf := createFixtureWithDownloadedLists(t,
		List{
			name: "list",
			path: "/list.yaml",
			md5:  "6ed1488a9d6558e66acab15964b4a91e",
			content: map[string]any{
				"allowed-licenses":    []string{"MIT"},
				"disallowed-licenses": []string{},
			},
		},
	)

	t.Run("correct md5", func(t *testing.T) {
		// from the test set up above, we know that we've downloaded one list
		// successfully and because we've made no changes, it should all be
		// correct
		err := sut.ValidateLocalLists()
		require.NoError(t, err)
	})

	t.Run("incorrect md5", func(t *testing.T) {
		// overwrite the list file with different content
		err := os.WriteFile(filepath.Join(conf.CacheDir, "list.yaml"), []byte("You will be like us"), 0644)
		require.NoError(t, err)

		err = sut.ValidateLocalLists()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "md5 mismatch for list 'list'")
	})

	t.Run("missing list", func(t *testing.T) {
		// delete list file
		err := os.Remove(filepath.Join(conf.CacheDir, "list.yaml"))
		require.NoError(t, err)

		err = sut.ValidateLocalLists()
		require.Error(t, err)
		assert.Contains(t, err.Error(), "list 'list' is missing")
	})
}

type List struct {
	name        string
	path        string
	md5         string
	content     map[string]any
	description string
	rating      float32
}

func createFixtureWithDownloadedLists(t *testing.T, lists ...List) (*curatedlists.Service, *helpers_test.MockServer, *config.Config) {
	server := newServerWithLists(t, lists...)
	t.Cleanup(func() {
		server.Close()
	})

	conf := &config.Config{
		CacheDir:           helpers_test.CreateTempDir(t, "license-checker-test-cache-*"),
		CuratedlistsSource: server.URL() + "/curated-lists.yaml",
	}

	sut := curatedlists.New(conf, http.DefaultTransport)
	err := sut.DownloadCuratedLists()
	require.NoError(t, err)

	return sut, server, conf
}

func newServerWithLists(t *testing.T, lists ...List) *helpers_test.MockServer {
	server := helpers_test.NewMockServer()

	listMetadata := make(map[string]any)
	for _, list := range lists {
		path := list.path
		if path == "" {
			path = fmt.Sprintf("%s.yaml", list.name)
		}

		url := fmt.Sprintf("%s/%s", server.URL(), path)

		hash := list.md5
		if hash == "" {
			// convert content to yaml and calculate md5
			if len(list.content) == 0 {
				list.content = make(map[string]any)
			}
			contentBytes, err := yaml.Marshal(list.content)
			require.NoError(t, err)

			hash = fmt.Sprintf("%x", md5.Sum(contentBytes))
		}

		listMetadata[list.name] = map[string]any{
			"url":         url,
			"md5":         hash,
			"description": list.description,
			"rating":      list.rating,
		}

		server.AddYamlResponse(url, list.content)
	}

	server.AddYamlResponse(server.URL()+"/curated-lists.yaml", listMetadata)

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
