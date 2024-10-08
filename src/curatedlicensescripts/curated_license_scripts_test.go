package curatedlicensescripts_test

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlicensescripts"
	filedownloader_test "github.com/eriklarko/license-checker/src/filedownloader/testhelpers"
	helpers_test "github.com/eriklarko/license-checker/src/helpers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHasScriptForPackageManager(t *testing.T) {
	availableScripts := []script{
		{
			Thing: &filedownloader_test.Thing{
				Name: "npm",
				Path: "/scripts/npm.sh",
			},
			ScriptContent: "echo 'HAH!'",
		},
	}

	t.Run("Downloads lock file if it doesn't exist", func(t *testing.T) {
		sut, httpServer, config := createServerEnvironmentWithScripts(t, availableScripts...)

		_, err := sut.HasScriptForPackageManager("npm")
		require.NoError(t, err)

		assert.Equal(t, 1, httpServer.GetHitCount("/metadata.yaml"))
		assert.FileExists(t, config.CacheDir+"/curated-license-scripts-lock.yaml")
	})

	t.Run("Does not download lock file if it exists", func(t *testing.T) {
		sut, httpServer, _ := createServerEnvironmentWithScripts(t, availableScripts...)

		err := sut.DownloadCuratedScripts()
		require.NoError(t, err)

		_, err = sut.HasScriptForPackageManager("npm")
		require.NoError(t, err)

		assert.Equal(t, 1, httpServer.GetHitCount("/metadata.yaml"))
	})

	t.Run("Returns true if script exists", func(t *testing.T) {
		sut, _, _ := createServerEnvironmentWithScripts(t, availableScripts...)

		exists, err := sut.HasScriptForPackageManager("npm")
		require.NoError(t, err)

		assert.True(t, exists)
	})

	t.Run("Returns false if script doesn't exists", func(t *testing.T) {
		sut, _, _ := createServerEnvironmentWithScripts(t, availableScripts...)

		exists, err := sut.HasScriptForPackageManager("foo")
		require.NoError(t, err)

		assert.False(t, exists)
	})
}

func TestDownloadScript(t *testing.T) {
	sut, httpServer, conf := createServerEnvironmentWithScripts(t, script{
		Thing: &filedownloader_test.Thing{
			Name: "script1",
			Path: "/some/deep/path/script1.sh",
		},
		ScriptContent: "echo 'script1'",
	})
	err := sut.DownloadCuratedScripts()
	require.NoError(t, err)

	t.Run("Download known script", func(t *testing.T) {
		// download the script
		path, err := sut.DownloadScript("script1")
		require.NoError(t, err)

		assert.Equal(t, filepath.Join(conf.CacheDir, "script1.sh"), path)
		// check that the script was downloaded correctly
		helpers_test.AssertFileExists(t, path, []byte("echo 'script1'"))
	})

	t.Run("Download unknown script", func(t *testing.T) {
		_, err := sut.DownloadScript("imaginary-script")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "no metadata")
		assert.Contains(t, err.Error(), "imaginary-script")
	})

	t.Run("Download known script twice", func(t *testing.T) {
		_, err := sut.DownloadScript("script1")
		require.NoError(t, err)

		_, err = sut.DownloadScript("script1")
		require.NoError(t, err)

		// check that the endpoint was only hit once
		assert.Equal(t, 1, httpServer.GetHitCount("/some/deep/path/script1.sh"))
	})
}

func TestSelectScript(t *testing.T) {
	scriptContent := "echo 'script1'"
	sut, _, conf := createServerEnvironmentWithScripts(t, script{
		Thing: &filedownloader_test.Thing{
			Name: "script1",
			Path: "/script1.sh",
		},
		ScriptContent: scriptContent,
	})
	err := sut.DownloadCuratedScripts()
	require.NoError(t, err)

	t.Run("valid script", func(t *testing.T) {
		err := sut.SelectScript("script1")
		require.NoError(t, err)

		// check that the list was downloaded
		helpers_test.AssertFileExists(t, filepath.Join(conf.CacheDir, "script1.sh"), []byte(scriptContent))
		// check that the list was selected
		assert.Equal(t, "script1", conf.SelectedCuratedScript)
	})

	t.Run("script does not exist", func(t *testing.T) {
		err := sut.SelectScript("non-existing-script")
		require.Error(t, err)
	})
}

type script struct {
	*filedownloader_test.Thing `yaml:",inline"`

	ScriptContent string `yaml:"-"`
}

func createServerEnvironmentWithScripts(t *testing.T, scripts ...script) (*curatedlicensescripts.Service, *helpers_test.MockServer, *config.Config) {
	for _, script := range scripts {
		if script.Content == nil {
			script.Content = bytes.NewReader([]byte(script.ScriptContent))
		}
	}
	server := filedownloader_test.NewServerWithThings(t, scripts...)
	t.Cleanup(func() {
		server.Close()
	})

	conf := &config.Config{
		CacheDir:             t.TempDir(),
		CuratedScriptsSource: server.URL() + "/metadata.yaml",
		Path:                 helpers_test.CreateTempFile(t, "config").Name(),
	}

	sut := curatedlicensescripts.New(conf)
	return sut, server, conf
}
