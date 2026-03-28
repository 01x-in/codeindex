package config

import (
	"os"
	"path/filepath"
)

// DetectionResult holds the result of language auto-detection.
type DetectionResult struct {
	Language string
	Marker   string
}

// DetectLanguages scans the given directory for project markers
// and returns detected languages.
func DetectLanguages(dir string) ([]DetectionResult, error) {
	markers := map[string]string{
		"package.json":    "typescript",
		"tsconfig.json":   "typescript",
		"go.mod":          "go",
		"pyproject.toml":  "python",
		"setup.py":        "python",
		"Cargo.toml":      "rust",
	}

	seen := map[string]bool{}
	var results []DetectionResult

	for marker, lang := range markers {
		path := filepath.Join(dir, marker)
		if _, err := os.Stat(path); err == nil {
			if !seen[lang] {
				seen[lang] = true
				results = append(results, DetectionResult{
					Language: lang,
					Marker:   marker,
				})
			}
		}
	}

	return results, nil
}
