package curatedlists

import (
	"fmt"
	"math"
	"net/url"
	"strings"

	"github.com/eriklarko/license-checker/src/config"
	"github.com/eriklarko/license-checker/src/curatedlicensescripts/packagemanagerdetector"
	"github.com/eriklarko/license-checker/src/filedownloader"
)

// ListMetadata defines the shape we expect the response from the curated lists
// endpoint to have.
// Example:
//
//	super-conservative-list:
//	  md5: 123
//	  url: https://example.com/super-conservative-list.yaml
//	less-conservative-list:
//	  md5: 456
//	  url: https://example.com/less-conservative-list.yaml
type ListMetadata map[string]ListInfo
type ListInfo struct {
	Md5 string `yaml:"md5"`
	Url string `yaml:"url"`

	Description string  `yaml:"description"`
	Rating      float32 `yaml:"rating,omitempty"`
}

func (l ListInfo) GetUrl() string {
	return l.Url
}

func (l ListInfo) GetMd5() string {
	return l.Md5
}

type Service struct {
	config         *config.Config
	fileDownloader *filedownloader.Service[ListInfo]
}

func New(
	config *config.Config,
) *Service {
	return &Service{
		config: config,
		fileDownloader: filedownloader.New[ListInfo](
			"curated-lists",
			config.CuratedListsSource,
			config.CacheDir,

			func(li ListInfo) (string, error) {
				url, err := url.Parse(li.Url)
				if err != nil {
					return "", fmt.Errorf("failed to parse url %s: %w", li.Url, err)
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

func (s *Service) DownloadCuratedLists() error {
	if s.config.CuratedListsSource == "" {
		return fmt.Errorf("no curated list source set, see README for how to configure this")
	}

	return s.fileDownloader.DownloadMetadata()
}

func (s *Service) GetAllLists() (ListMetadata, error) {
	lists, err := s.fileDownloader.GetLockFileContents()
	if err != nil {
		return nil, fmt.Errorf("failed to read list lock file: %w", err)
	}

	return lists, nil
}

// GetHighlyRatedList returns the name and description of the list with the
// highest rating.
//
// NOTE: The behavior is undefined if there are multiple lists with the same
// rating
func (s *Service) GetHighlyRatedList() (string, string, error) {
	lists, err := s.fileDownloader.GetLockFileContents()
	if err != nil {
		return "", "", fmt.Errorf("failed to read list lock file: %w", err)
	}

	// find highest rated list
	var bestListName string
	var bestListInfo ListInfo
	var bestRating float32 = math.SmallestNonzeroFloat32
	for listName, listInfo := range lists {
		if listInfo.Rating > bestRating {
			bestListName = listName
			bestListInfo = listInfo

			bestRating = listInfo.Rating
		}
	}

	return bestListName, bestListInfo.Description, nil
}

func (s *Service) DownloadList(listName string) error {
	// No need to download the script if it already exists
	path, err := s.fileDownloader.GetDestinationPath(listName)
	if err != nil {
		return fmt.Errorf("failed to get destination path: %w", err)
	}

	exists, err := packagemanagerdetector.FileExists(path)
	if err != nil {
		return fmt.Errorf("failed to check if script exists: %w", err)
	}

	if !exists {
		err = s.fileDownloader.Download(listName)
		if err != nil {
			return fmt.Errorf("failed to download script: %w", err)
		}
	}

	return nil
}

func (s *Service) SelectList(listName string) error {
	err := s.DownloadList(listName)
	if err != nil {
		return fmt.Errorf("failed to download list: %w", err)
	}

	s.config.SelectedCuratedList = listName
	return s.config.Write()
}
