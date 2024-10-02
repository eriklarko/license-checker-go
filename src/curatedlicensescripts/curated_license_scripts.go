package curatedlicensescripts

import (
	"fmt"

	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlicensescripts/packagemanagerdetector"
	"github.com/eriklarko/license-checker/src/filedownloader"
)

type ScriptMetadata map[string]ScriptInfo
type ScriptInfo struct {
	Md5         string `yaml:"md5"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`
}

func (l ScriptInfo) GetUrl() string {
	return l.Url
}

func (l ScriptInfo) GetMd5() string {
	return l.Md5
}

type Service struct {
	config         *config.Config
	fileDownloader *filedownloader.Service[ScriptInfo]
}

func New(
	config *config.Config,
) *Service {
	return &Service{
		config: config,
		fileDownloader: filedownloader.New[ScriptInfo](
			"curated-license-scripts",
			config.CuratedScriptsSource,
			config.CacheDir,
		),
	}
}

func (s *Service) DownloadCuratedScripts() error {
	if s.config.CuratedScriptsSource == "" {
		return fmt.Errorf("no curated script source set, see README for how to configure this")
	}

	return s.fileDownloader.DownloadMetadata()
}

func (s *Service) DownloadScript(packageManager string) (string, error) {
	path := s.fileDownloader.GetDestinationPath(packageManager)

	// No need to download the script if it already exists
	exists, err := packagemanagerdetector.FileExists(path)
	if err != nil {
		return "", fmt.Errorf("failed to check if script for package manager '%s' exists: %w", packageManager, err)
	}

	if !exists {
		err = s.fileDownloader.Download(packageManager)
		if err != nil {
			return "", fmt.Errorf("failed to download script for package manager '%s': %w", packageManager, err)
		}
	}

	return path, nil
}
