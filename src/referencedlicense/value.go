package referencedlicense

import (
	"crypto/md5"
	"encoding/csv"
	"fmt"
	"os"
	"path/filepath"
)

type ReferenceValueVerifier struct {
	outputFile string

	knownLicenses map[string]string

	outputFileHandle *os.File
	writer           *csv.Writer
}

func NewReferenceValueVerifier(outputFile string) *ReferenceValueVerifier {
	return &ReferenceValueVerifier{
		outputFile: outputFile,
	}
}

func (r *ReferenceValueVerifier) Record(dependency, license string) error {
	md5Sum, err := calculateMD5Sum(license)
	if err != nil {
		return err
	}

	err = r.appendToLicenseFile(dependency, md5Sum)
	if err != nil {
		return err
	}

	return nil
}

func calculateMD5Sum(license string) (string, error) {
	hash := md5.Sum([]byte(license))
	md5Sum := fmt.Sprintf("%x", hash)
	return md5Sum, nil
}

func (r *ReferenceValueVerifier) appendToLicenseFile(dependency, md5Sum string) error {
	if r.writer == nil {
		var err error
		r.writer, err = r.openWriter()
		if err != nil {
			return err
		}
	}

	record := []string{dependency, md5Sum}
	err := r.writer.Write(record)
	if err != nil {
		return err
	}

	return nil
}

func (r *ReferenceValueVerifier) openWriter() (*csv.Writer, error) {
	absPath, err := filepath.Abs(r.outputFile)
	if err != nil {
		return nil, err
	}

	file, err := os.OpenFile(absPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	r.outputFileHandle = file

	writer := csv.NewWriter(file)
	return writer, nil
}

func (r *ReferenceValueVerifier) Close() error {
	if r.writer != nil {
		r.writer.Flush()
		r.writer = nil
	}

	if r.outputFileHandle != nil {
		err := r.outputFileHandle.Close()
		if err != nil {
			return err
		}
	}

	return nil
}
func (r *ReferenceValueVerifier) CheckLicenseMatch(license string) (bool, error) {
	md5Sum, err := calculateMD5Sum(license)
	if err != nil {
		return false, err
	}

	if r.knownLicenses == nil {
		err := r.loadKnownLicenses()
		if err != nil {
			return false, err
		}
	}

	for _, record := range r.knownLicenses {
		if record == md5Sum {
			return true, nil
		}
	}

	return false, nil
}

func (r *ReferenceValueVerifier) loadKnownLicenses() error {
	absPath, err := filepath.Abs(r.outputFile)
	if err != nil {
		return err
	}

	file, err := os.Open(absPath)
	if err != nil {
		return err
	}
	defer file.Close()

	reader := csv.NewReader(file)
	records, err := reader.ReadAll()
	if err != nil {
		return err
	}

	r.knownLicenses = make(map[string]string)
	for _, record := range records {
		dependency := record[0]
		md5Sum := record[1]
		r.knownLicenses[dependency] = md5Sum
	}

	return nil
}
