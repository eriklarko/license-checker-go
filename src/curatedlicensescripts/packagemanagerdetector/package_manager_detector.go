package packagemanagerdetector

import (
	"fmt"
)

type Service struct {
	// Where to look for package manager files
	Directory string
}

func New(directory string) *Service {
	return &Service{Directory: directory}
}

func (s *Service) FindLikelyPackageManagers() ([]string, error) {
	// if file exists, return the package manager
	filesToPackageManager := map[string]string{
		"npm":        "package.json",
		"go modules": "go.mod",
		"pip":        "requirements.txt",
		"maven":      "pom.xml",
		"gradle":     "build.gradle",
	}

	var detectedPackageManagers []string
	for packageManager, file := range filesToPackageManager {
		exists, err := FileExists(fmt.Sprintf("%s/%s", s.Directory, file))
		if err != nil {
			return nil, err
		}

		if exists {
			detectedPackageManagers = append(detectedPackageManagers, packageManager)
		}
	}

	return detectedPackageManagers, nil
}
