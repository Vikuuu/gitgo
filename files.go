package gitgo

import (
	"fmt"
	"io/fs"
	"path/filepath"
)

// Returns the flatten directory structure
func ListFiles(dir string) ([]string, error) {
	var workfiles []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("from listFiles %s", err)
		}
		name := filepath.Base(path)
		// skip the files or directories found in the ignore hashmap
		if _, found := g_ignore[name]; found {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		// Append only files, not directories
		if !d.IsDir() {
			relPath, _ := filepath.Rel(dir, path)
			workfiles = append(workfiles, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return workfiles, nil
}
