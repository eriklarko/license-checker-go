package packagemanagerdetector_test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/eriklarko/license-checker/src/curatedlicensescripts/packagemanagerdetector"
)

func BenchmarkDetect_NoManagers(b *testing.B) {
	detector := packagemanagerdetector.New(b.TempDir())

	runBenchmarks(b, detector)
}

func BenchmarkDetect_OneManager(b *testing.B) {
	dir := b.TempDir()
	touchFile(dir, "package.json")

	detector := packagemanagerdetector.New(dir)
	runBenchmarks(b, detector)
}

func BenchmarkDetect_AllManagers(b *testing.B) {
	dir := b.TempDir()
	touchFile(dir, "package.json")
	touchFile(dir, "go.mod")
	touchFile(dir, "requirements.txt")
	touchFile(dir, "pom.xml")
	touchFile(dir, "build.gradle")

	detector := packagemanagerdetector.New(dir)
	runBenchmarks(b, detector)
}

func BenchmarkDetect_SmallNumFilesInDir(b *testing.B) {
	dir := b.TempDir()
	// add one valid file just for kicks
	touchFile(dir, "package.json")
	for i := 0; i < 50; i++ {
		touchFile(dir, fmt.Sprintf("file-%d", i))
	}

	detector := packagemanagerdetector.New(dir)
	runBenchmarks(b, detector)
}

func BenchmarkDetect_LoadsOfFilesInDir(b *testing.B) {
	dir := b.TempDir()
	// add one valid file just for kicks
	touchFile(dir, "package.json")
	for i := 0; i < 1000; i++ {
		touchFile(dir, fmt.Sprintf("file-%d", i))
	}

	detector := packagemanagerdetector.New(dir)
	runBenchmarks(b, detector)
}

var result []string

func runBenchmarks(b *testing.B, detector *packagemanagerdetector.Service) {
	b.Run("probing for individual files", func(b *testing.B) {
		var r []string
		for i := 0; i < b.N; i++ {
			detector.FindLikelyPackageManagers()
		}
		result = r
	})
	b.Run("looping through all files to see which ones exist", func(b *testing.B) {
		var r []string
		for i := 0; i < b.N; i++ {
			otherImplOfDetect(detector.Directory)
		}
		result = r
	})
}

func touchFile(dir, fileName string) {
	path := filepath.Join(dir, fileName)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		f, err := os.Create(path)
		if err != nil {
			panic(err)
		}
		f.Close()
	}
}

func otherImplOfDetect(dir string) ([]string, error) {
	var packageManagers []string

	files, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		switch strings.ToLower(file.Name()) {
		case "package.json":
			packageManagers = append(packageManagers, "npm")
		case "go.mod":
			packageManagers = append(packageManagers, "go")
		case "requirements.txt":
			packageManagers = append(packageManagers, "pip")
		case "pom.xml":
			packageManagers = append(packageManagers, "maven")
		case "build.gradle":
			packageManagers = append(packageManagers, "gradle")
		}
	}

	return packageManagers, nil
}
