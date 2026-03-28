package config

import "fmt"

// ValidateSchema performs deep validation of a config,
// checking all fields for correctness beyond basic type checking.
func ValidateSchema(cfg Config) []string {
	var errors []string

	if cfg.Version < 1 {
		errors = append(errors, "version must be >= 1")
	}

	if len(cfg.Languages) == 0 {
		errors = append(errors, "at least one language must be configured")
	}

	if cfg.IndexPath == "" {
		errors = append(errors, "index_path must not be empty")
	}

	for _, prim := range cfg.QueryPrimitives {
		if !isValidPrimitive(prim) {
			errors = append(errors, fmt.Sprintf("unknown query primitive: %q", prim))
		}
	}

	return errors
}

func isValidPrimitive(p string) bool {
	valid := map[string]bool{
		"get_file_structure": true,
		"find_symbol":        true,
		"get_references":     true,
		"get_callers":        true,
		"get_subgraph":       true,
		"reindex":            true,
	}
	return valid[p]
}
