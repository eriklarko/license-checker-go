package filedownloader

import (
	"bytes"
	"crypto/md5"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"gopkg.in/yaml.v3"
)

type Metadata struct {
	Url string `yaml:"url"`
	Md5 string `yaml:"md5"`
}

type Service struct {
	// defines what kind of content we are downloading; curated licese lists,
	// package manager scripts etc
	contentType string

	// the endpoint where the list of available items is fetched from
	// It's expected to return a yaml file with a map of item names to metadata
	// about the item.
	//
	// Example:
	//   ```
	//   list1:
	//     url: http://example.com/list1.yaml
	//     md5: 73411061536ff8a32777eec043ece0e6
	//   list2:
	//     url: http://example.com/list2.yaml
	//     md5: fad50251071a2532729e7f4beb79f8ca
	//   ```
	metadataURL string

	// the directory where the downloaded files are stored
	downloadDir string

	// makes sure only one goroutine reads the lock file at a time
	lockFileReadLock sync.RWMutex
	// cache the contents of the list lock file
	lockFile map[string]Metadata // TODO: type extends Metadata
}

// New creates a new instance of the file downloader service
//
// Example:
//
//	```
//	downloader := filedownloader.New("curated-lists", "http://example.com/metadata.yaml", "/tmp")
//	```
func New(contentType, metadataURL, downloadDir string) *Service {
	return &Service{
		contentType: contentType,
		metadataURL: metadataURL,
		downloadDir: downloadDir,
	}
}

func (s *Service) DownloadMetadata() error {
	slog.Info("Fetching metadata", "endpoint", s.metadataURL, "content_type", s.contentType)

	// Fetch the list of curated files
	body, err := s.executeGetRequest(s.metadataURL)
	if err != nil {
		return fmt.Errorf("failed to fetch latest metadata from %s: %w", s.metadataURL, err)
	}
	defer body.Close()

	// set up body to be read twice. First just read like normal, then the body
	// is available in `buf`. This allows us to parse the body and log it in
	// case of errors
	buf := bytes.Buffer{}
	body = io.NopCloser(io.TeeReader(body, &buf))

	// parse the response body
	var parsedBody map[string]Metadata
	err = yaml.NewDecoder(body).Decode(&parsedBody)
	if err != nil {
		slog.Debug(
			"failed to parse response body",
			"content_type", s.contentType,
			"error", err,
			"body", string(truncate(buf.Bytes(), 100)),
		)
		return fmt.Errorf("failed to parse response body: %w", err)
	}

	err = s.writeLockFile(parsedBody)
	if err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

func (s *Service) executeGetRequest(endpoint string) (io.ReadCloser, error) {
	resp, err := http.DefaultClient.Get(endpoint)
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

func truncate[T any](s []T, maxLength int) []T {
	if len(s) <= maxLength {
		return s
	}
	return s[:maxLength]
}

func (s *Service) writeLockFile(lists map[string]Metadata) error {
	s.lockFileReadLock.Lock()
	defer s.lockFileReadLock.Unlock()

	// Ensure the working directory exists
	if err := os.MkdirAll(s.downloadDir, os.ModePerm); err != nil {
		return fmt.Errorf("failed to create cache directory %s: %w", s.downloadDir, err)
	}

	path := s.GetLockFilePath()
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	err = yaml.NewEncoder(file).Encode(lists)
	if err != nil {
		return fmt.Errorf("failed to write lock file: %w", err)
	}

	return nil
}

func (s *Service) GetLockFileContents() (map[string]Metadata, error) {
	s.lockFileReadLock.RLock()
	defer s.lockFileReadLock.RUnlock()

	if s.lockFile != nil {
		return s.lockFile, nil
	}

	path := s.GetLockFilePath()
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open file %s: %w", path, err)
	}
	defer file.Close()

	var lists map[string]Metadata
	err = yaml.NewDecoder(file).Decode(&lists)
	if err != nil {
		return nil, fmt.Errorf("failed to decode file: %w", err)
	}

	// cache contents for future use
	s.lockFile = lists
	return s.lockFile, nil
}

func (s *Service) GetDownloadDir() string {
	return s.downloadDir
}

func (s *Service) GetLockFilePath() string {
	return filepath.Join(s.downloadDir, s.contentType+"-lock.yaml")
}

// Download downloads a file from the internet and stores it in the download directory
// The `name` parameter is the key in the lock file that corresponds to the file to download
func (s *Service) Download(name string) error {
	metadatas, err := s.GetLockFileContents() // data is already plural, but you can pluralize it again
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	metadata, found := metadatas[name]
	if !found {
		return fmt.Errorf("no metadata found for '%s'", name)
	}
	slog.Info("Downloading list", "name", name, "url", metadata.Url)

	body, err := s.executeGetRequest(metadata.Url)
	if err != nil {
		return fmt.Errorf("failed to download %s from %s: %w", name, metadata.Url, err)
	}
	defer body.Close()

	// read whole list into memory, calculate md5 sum and if it's correct, write
	// the file to disk
	bodyBytes, err := io.ReadAll(body)
	if err != nil {
		return fmt.Errorf("failed to read list %s: %w", name, err)
	}

	calculatedMd5 := fmt.Sprintf("%x", md5.Sum(bodyBytes))
	if calculatedMd5 != metadata.Md5 {
		return fmt.Errorf("md5 mismatch for item %s: expected %s, got %s", name, metadata.Md5, calculatedMd5)
	}

	err = s.writeToDisk(name, bodyBytes)
	if err != nil {
		return fmt.Errorf("failed to write item %s to disk: %w", name, err)
	}

	return nil
}

func (s *Service) writeToDisk(itemName string, body []byte) error {
	path := filepath.Join(s.downloadDir, itemName+".yaml")
	file, err := os.Create(path)
	if err != nil {
		return fmt.Errorf("failed to create file %s: %w", path, err)
	}
	defer file.Close()

	_, err = file.Write(body)
	if err != nil {
		return fmt.Errorf("failed to write item %s to disk: %w", itemName, err)
	}

	return nil
}

func (s *Service) TestValidateDownloadedFiles() error {
	// read list lock file
	metadata, err := s.GetLockFileContents()
	if err != nil {
		return fmt.Errorf("failed to read lock file: %w", err)
	}

	// check that all downloaded files the correct md5 sum
	var errs []error
	for itemName, metadatum := range metadata {
		path := filepath.Join(s.downloadDir, itemName+".yaml")

		// check if file exists
		_, err := os.Stat(path)
		if errors.Is(err, os.ErrNotExist) {
			// the user hasn't downloaded this thing yet
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
		if calculatedMd5 != metadatum.Md5 {
			errs = append(errs, fmt.Errorf("md5 mismatch for file '%s': expected %s, got %s", path, metadatum.Md5, calculatedMd5))
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
