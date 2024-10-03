package curatedlists_test

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlists"
	filedownloader_test "github.com/eriklarko/license-checker/src/filedownloader/testhelpers"
	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestDownloadList(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	list1Content := map[string]any{
		"allowed-licenses":    []string{"MIT"},
		"disallowed-licenses": []string{"GPL-3.0"},
	}
	sut, httpServer, conf := createServerEnvironmentWithLists(t,
		list{
			Thing: &filedownloader_test.Thing{
				Name: "list1",
				Path: "/some/deep/path/list1.yaml",
			},
			Description: "list1 description",
			YamlContent: list1Content,
		},
	)

	t.Run("download known list", func(t *testing.T) {
		err := sut.DownloadList("list1")
		require.NoError(t, err)

		// check that the list was downloaded
		helpers_test.AssertYamlFileExists(t, filepath.Join(conf.CacheDir, "list1.yaml"), list1Content)
	})

	t.Run("download unknown list", func(t *testing.T) {
		err := sut.DownloadList("unknown-list")
		require.Error(t, err)

		assert.Contains(t, err.Error(), "no metadata")
		assert.Contains(t, err.Error(), "unknown-list")
	})

	t.Run("Download known list twice", func(t *testing.T) {
		err := sut.DownloadList("list1")
		require.NoError(t, err)

		err = sut.DownloadList("list1")
		require.NoError(t, err)

		// check that the endpoint was only hit once
		assert.Equal(t, 1, httpServer.GetHitCount("/some/deep/path/list1.yaml"))
	})
}

func TestGetAllLists(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	sut, httpMock, _ := createServerEnvironmentWithLists(t,
		list{
			Thing: &filedownloader_test.Thing{
				Name: "list1",
				Path: "/list1.yaml",
				Md5:  "73411061536ff8a32777eec043ece0e6",
			},
			Description: "list1 description",
			YamlContent: map[string]any{
				"allowed-licenses":    []string{"MIT"},
				"disallowed-licenses": []string{"GPL-3.0"},
			},
		},
		list{
			Thing: &filedownloader_test.Thing{
				Name: "list2",
				Path: "/list2.yaml",
				Md5:  "fad50251071a2532729e7f4beb79f8ca",
			},
			Description: "list2 description",
			YamlContent: map[string]any{
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
			Description: "list1 description",
		},
		"list2": curatedlists.ListInfo{
			Md5:         "fad50251071a2532729e7f4beb79f8ca",
			Url:         httpMock.URL() + "/list2.yaml",
			Description: "list2 description",
		},
	}, lists)
}

func TestGetHighestRatedList(t *testing.T) {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	testCases := map[string]struct {
		lists            []list
		expectedListName string
	}{
		"empty list": {
			lists:            nil,
			expectedListName: "",
		},
		"single list": {
			lists: []list{
				{
					Thing: &filedownloader_test.Thing{
						Name: "list1",
					},
					Description: "list1 description",
					Rating:      0.5,
				},
			},
			expectedListName: "list1",
		},
		"multiple lists": {
			lists: []list{
				{
					Thing: &filedownloader_test.Thing{
						Name: "list1",
					},
					Description: "list1 description",
					Rating:      0.5,
				},
				{
					Thing: &filedownloader_test.Thing{
						Name: "list2",
					},
					Description: "list2 description",
					Rating:      0.9,
				},
				{
					Thing: &filedownloader_test.Thing{
						Name: "list3",
					},
					Description: "list3 description",
					Rating:      0.7,
				},
			},
			expectedListName: "list2",
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			sut, _, _ := createServerEnvironmentWithLists(t, tc.lists...)

			var expectedList list
			if len(tc.lists) == 0 {
				// if there are no lists, the we expect "" for name and description
				expectedList = list{
					Thing: &filedownloader_test.Thing{
						Name: "",
					},
					Description: "",
				}
			} else {

				// to avoid having to specify both list name and description in
				// the test case, we infer the expected description here by
				// searching for the list with the expected name
				var found bool
				expectedList, found = lo.Find(tc.lists, func(list list) bool {
					return list.Name == tc.expectedListName
				})
				require.True(t, found, "expected list not found in test case")
			}

			listName, listDescription, err := sut.GetHighlyRatedList()
			require.NoError(t, err)
			assert.Equal(t, expectedList.Name, listName)
			assert.Equal(t, expectedList.Description, listDescription)
		})
	}
}

func TestSelectList(t *testing.T) {
	listContent := map[string]any{
		"allowed-licenses":    []string{"MIT"},
		"disallowed-licenses": []string{"GPL-3.0"},
	}
	sut, _, conf := createServerEnvironmentWithLists(t,
		list{
			Thing: &filedownloader_test.Thing{
				Name: "list1",
				Path: "/list1.yaml",
			},
			Description: "list1 description",
			YamlContent: listContent,
		},
	)

	t.Run("valid list", func(t *testing.T) {
		err := sut.SelectList("list1")
		require.NoError(t, err)

		// check that the list was downloaded
		helpers_test.AssertYamlFileExists(t, filepath.Join(conf.CacheDir, "list1.yaml"), listContent)
		// check that the list was selected
		assert.Equal(t, "list1", conf.SelectedCuratedList)
	})

	t.Run("list does not exist", func(t *testing.T) {
		err := sut.SelectList("non-existing-list")
		require.Error(t, err)
	})
}

type list struct {
	*filedownloader_test.Thing `yaml:",inline"`

	YamlContent map[string]any `yaml:"-"`

	Description string  `yaml:"description"`
	Rating      float32 `yaml:"rating"`
}

func createServerEnvironmentWithLists(t *testing.T, lists ...list) (*curatedlists.Service, *helpers_test.MockServer, *config.Config) {
	for _, list := range lists {
		if list.Content == nil {
			yamlBody, err := yaml.Marshal(list.YamlContent)
			require.NoError(t, err)

			list.Content = bytes.NewReader(yamlBody)
		}
	}

	server := filedownloader_test.NewServerWithThings(t, lists...)
	t.Cleanup(func() {
		server.Close()
	})

	conf := &config.Config{
		CacheDir:           t.TempDir(),
		CuratedListsSource: server.URL() + "/metadata.yaml",
		Path:               helpers_test.CreateTempFile(t, "config").Name(),
	}

	sut := curatedlists.New(conf)
	err := sut.DownloadCuratedLists()
	require.NoError(t, err)

	return sut, server, conf
}
