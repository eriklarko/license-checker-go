package curatedlicensescripts

import (
	"fmt"
	"net/url"
	"strings"

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
			func(si ScriptInfo) (string, error) {
				url, err := url.Parse(si.Url)
				if err != nil {
					return "", fmt.Errorf("failed to parse url %s: %w", si.Url, err)
				}

				pathSegments := strings.Split(url.Path, "/")
				if len(pathSegments) == 0 {
					return url.Path, nil
				}
				return pathSegments[len(pathSegments)-1], nil
			},
		),
	}
}

func (s *Service) HasScriptForPackageManager(packageManager string) (bool, error) {
	lockFileExists, err := packagemanagerdetector.FileExists(s.fileDownloader.GetLockFilePath())
	if err != nil {
		return false, fmt.Errorf("failed to check if lock file exists: %w", err)
	}
	if !lockFileExists {
		err := s.DownloadCuratedScripts()
		if err != nil {
			return false, fmt.Errorf("failed to download curated scripts: %w", err)
		}
	}

	scripts, err := s.fileDownloader.GetLockFileContents()
	if err != nil {
		return false, fmt.Errorf("failed to read lock file: %w", err)
	}

	for script := range scripts {
		if script == packageManager {
			return true, nil
		}
	}
	return false, nil
}

func (s *Service) DownloadCuratedScripts() error {
	if s.config.CuratedScriptsSource == "" {
		return fmt.Errorf("no curated script source set, see README for how to configure this")
	}

	return s.fileDownloader.DownloadMetadata()
}

func (s *Service) DownloadScript(packageManager string) (string, error) {
	// No need to download the script if it already exists
	path, err := s.fileDownloader.GetDestinationPath(packageManager)
	if err != nil {
		return "", fmt.Errorf("failed to get destination path: %w", err)
	}

	exists, err := packagemanagerdetector.FileExists(path)
	if err != nil {
		return "", fmt.Errorf("failed to check if script exists: %w", err)
	}

	if !exists {
		err = s.fileDownloader.Download(packageManager)
		if err != nil {
			return "", fmt.Errorf("failed to download script: %w", err)
		}
	}

	return path, nil
}

func (s *Service) SelectScript(packageManager string) error {
	path, err := s.DownloadScript(packageManager)
	if err != nil {
		return fmt.Errorf("failed to download script: %w", err)
	}

	s.config.LicensesScript = path
	s.config.SelectedCuratedScript = packageManager
	return s.config.Write()
}
