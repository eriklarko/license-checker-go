package curatedlists

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"math"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/eriklarko/license-checker/src/config"
	"gopkg.in/yaml.v3"
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
	Md5         string `yaml:"md5"`
	Url         string `yaml:"url"`
	Description string `yaml:"description"`

	Rating float32 `yaml:"rating,omitempty"`
}

type Service struct {
	config       *config.Config
	roundTripper http.RoundTripper

	// makes sure only one goroutine reads the lock file at a time
	lockFileReadLock sync.Mutex
	// cache the contents of the list lock file
	lockFile ListMetadata
}

// New creates a new Service using the provided http.RoundTripper to make requests.
// Usage:
//
//	import "net/http"
//	var s = curatedlists.New(config, http.DefaultTransport)
func New(
	config *config.Config,
	roundTripperd http.RoundTripper,
) *Service {
	return &Service{
		config:       config,
		roundTripper: roundTripperd,
	}
}

func (s *Service) DownloadCuratedLists() error {
	if s.config.CuratedlistsSource == "" {
		return fmt.Errorf("no curated list source set, see README for how to configure this")
	}

	endpoint := s.config.CuratedlistsSource
	slog.Info("Fetching curated lists", "endpoint", endpoint)

	// Fetch the list of curated files
	body, err := s.executeGetRequest(endpoint)
	if err != nil {
		return fmt.Errorf("failed to fetch latest lists from %s: %w", endpoint, err)
	}
	defer body.Close()

	// set up body to be read twice. First just read like normal, then the body
	// is available in `buf`. This allows us to parse the body and log it in
	// case of errors
	buf := bytes.Buffer{}
	body = io.NopCloser(io.TeeReader(body, &buf))

	// parse the response body
	var parsedBody ListMetadata
	err = yaml.NewDecoder(body).Decode(&parsedBody)
	if err != nil {
		slog.Debug(
			"failed to parse response body",
			"error", err,
			"body", string(truncate(buf.Bytes(), 100)),
		)
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	err = s.writeListLockFile(parsedBody)
	if err != nil {
		return fmt.Errorf("failed to write list lock file: %w", err)
	}

	err = s.downloadLists()
	if err != nil {
		return fmt.Errorf("failed to download lists: %w", err)
	}

	return nil

}

func (s *Service) executeGetRequest(endpoint string) (io.ReadCloser, error) {
	request, err := createRequestObject(endpoint)
	if err != nil {
		return nil, fmt.Errorf("failed to create HTTP request object: %w", err)
	}

	resp, err := s.roundTripper.RoundTrip(request)
	if err != nil {
		return nil, fmt.Errorf("failed to execute HTTP request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		// Read the body to include it in the error message. ignore error as this is best effort
		body, err := io.ReadAll(resp.Body)
		slog.Debug("failed to read response body", "error", err, "endpoint", endpoint)

		return nil, fmt.Errorf("received status code %d: body: %s", resp.StatusCode, truncate(body, 100))
	}

	return resp.Body, nil
}

func createRequestObject(url string) (*http.Request, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	return req, nil
}

func truncate[T any](s []T, maxLength int) []T {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength]
}

func (s *Service) writeListLockFile(lists ListMetadata) error {

	// Ensure the cache directory exists
	if err := os.MkdirAll(s.config.CacheDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", s.config.CacheDir, err)
	}

	path := s.GetLockFilePath()
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	err = yaml.NewEncoder(file).Encode(lists)
	if err != nil {
		return fmt.Errorf("failed to write list lock file: %w", err)
	}

	return nil
}

func (s *Service) downloadLists() error {
	lists, err := s.readListLockFile()
	if err != nil {
		return fmt.Errorf("failed to read list lock file: %w", err)
	}

	// download all lists concurrently
	var wg sync.WaitGroup
	errCh := make(chan error, len(lists))
	defer close(errCh)
	for listName, listInfo := range lists {
		wg.Add(1)

		go func(listName string, listInfo ListInfo) {
			defer wg.Done()

			err := s.downloadList(listName, listInfo)
			if err != nil {
				errCh <- fmt.Errorf("failed to download list %s: %w", listName, err)
			}

		}(listName, listInfo)
	}
	wg.Wait()

	// combine errors
	errMsgs := make([]string, 0, len(errCh))
	for len(errCh) > 0 {
		err := <-errCh
		errMsgs = append(errMsgs, err.Error())
	}
	if len(errMsgs) > 0 {
		return fmt.Errorf("multiple errors occurred: %s", strings.Join(errMsgs, "; "))
	}
	return nil
}

func (s *Service) readListLockFile() (ListMetadata, error) {
	s.lockFileReadLock.Lock()
	defer s.lockFileReadLock.Unlock()

	if s.lockFile != nil {
		return s.lockFile, nil
	}

	path := s.GetLockFilePath()
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var lists ListMetadata
	err = yaml.NewDecoder(file).Decode(&lists)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}

	// cache contents for future use
	s.lockFile = lists
	return s.lockFile, nil
}

func (s *Service) GetLockFilePath() string {
	return filepath.Join(s.config.CacheDir, "list-lock.yaml")
}

// downloadList downloads a single list from the provided URL and saves it to
// disk, including verifying md5 sum with value from server
func (s *Service) downloadList(listName string, listInfo ListInfo) error {
	slog.Info("Downloading list", "name", listName, "url", listInfo.Url)

	body, err := s.executeGetRequest(listInfo.Url)
	if err != nil {
		return fmt.Errorf("failed to download list %s from %s: %w", listName, listInfo.Url, err)
	}
	defer body.Close()

	// read whole list into memory, calculate md5 sum and if it's correct, write
	// the file to disk
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read list %s: %w", listName, err)
	}

	calculatedMd5 := fmt.Sprintf("%x", md5.Sum(bodyBytes))
	if calculatedMd5 != listInfo.Md5 {
		return fmt.Errorf("md5 mismatch for list %s: expected %s, got %s", listName, listInfo.Md5, calculatedMd5)
	}

	err = s.writeListToDisk(listName, bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to write list %s to disk: %w", listName, err)
	}

	return nil
}

func (s *Service) writeListToDisk(listName string, body []byte) error {
	path := filepath.Join(s.config.CacheDir, listName+".yaml")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write list %s to disk: %w", listName, err)
	}

	return nil
}

func (s *Service) ValidateLocalLists() error {
	// read list lock file
	lists, err := s.readListLockFile()
	if err != nil {
		return fmt.Errorf("failed to read list lock file: %w", err)
	}

	// check if all lists are present and have correct md5 sum
	var errs []error
	for listName, listInfo := range lists {
		path := filepath.Join(s.config.CacheDir, listName+".yaml")

		// check if file exists
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			errs = append(errs, fmt.Errorf("list '%s' is missing: %w", listName, err))
			continue
		} else if err != nil {
			errs = append(errs, fmt.Errorf("failed to check if file %s exists: %w", path, err))
			continue
		}

		// read file into memory
		file, err := os.Open(path)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to open file %s: %w", path, err))
			continue
		}
		defer file.Close()

		fileContents, err := io.ReadAll(file)
		if err != nil {
			errs = append(errs, fmt.Errorf("failed to read file %s: %w", path, err))
			continue
		}

		// verify md5 sum
		calculatedMd5 := fmt.Sprintf("%x", md5.Sum(fileContents))
		if calculatedMd5 != listInfo.Md5 {
			errs = append(errs, fmt.Errorf("md5 mismatch for list '%s': expected %s, got %s", listName, listInfo.Md5, calculatedMd5))
			continue
		}
	}

	if len(errs) == 0 {
		return nil
	} else if len(errs) == 1 {
		return errs[0]
	} else {
		errMsgs := make([]string, len(errs))
		for i, err := range errs {
			errMsgs[i] = err.Error()
		}
		return fmt.Errorf("multiple errors occurred: %s", strings.Join(errMsgs, "; "))
	}
}

func (s *Service) GetAllLists() (ListMetadata, error) {
	lists, err := s.readListLockFile()
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
	lists, err := s.readListLockFile()
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

func (s *Service) SelectList(listName string) error {
	return s.config.PersistCuratedListChoice(listName)
}
