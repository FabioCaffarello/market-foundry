package migrate

import (
	"fmt"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
)

// filenamePattern matches {NNN}_{action}_{target}.sql
var filenamePattern = regexp.MustCompile(`^(\d{3})_.+\.sql$`)

// ReadCatalog reads all .sql files from dir, validates naming, and returns
// them sorted by version number in ascending order.
func ReadCatalog(dir string) ([]Migration, error) {
	matches, err := filepath.Glob(filepath.Join(dir, "*.sql"))
	if err != nil {
		return nil, fmt.Errorf("glob migrations: %w", err)
	}

	migrations := make([]Migration, 0, len(matches))
	seen := make(map[uint32]string)

	for _, path := range matches {
		base := filepath.Base(path)
		m := filenamePattern.FindStringSubmatch(base)
		if m == nil {
			return nil, fmt.Errorf("invalid migration filename %q: must match {NNN}_{action}_{target}.sql", base)
		}

		ver, err := strconv.ParseUint(m[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("parse version from %q: %w", base, err)
		}

		version := uint32(ver)
		name := base[:len(base)-len(".sql")] // strip .sql extension

		if prev, ok := seen[version]; ok {
			return nil, fmt.Errorf("duplicate migration version %03d: %q and %q", version, prev, name)
		}
		seen[version] = name

		migrations = append(migrations, Migration{
			Version: version,
			Name:    name,
			Path:    path,
		})
	}

	slices.SortFunc(migrations, func(a, b Migration) int {
		if a.Version < b.Version {
			return -1
		}
		if a.Version > b.Version {
			return 1
		}
		return 0
	})

	return migrations, nil
}
