package gitgo

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

var ErrMissingFile = errors.New("no file with the name")

// Returns the flatten directory structure
func ListFiles(dir string, rootPath string) ([]string, error) {
	var workfiles []string

	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("from WalkDir %s", err)
		}

		// check if the given dir string is file or dir ?
		s, err := os.Stat(path)
		// if file is not present
		if os.IsNotExist(err) {
			return fmt.Errorf("%w '%s'", ErrMissingFile, dir)
		}
		if !s.IsDir() {
			relPath, err := filepath.Rel(dir, path)
			if err != nil {
				return fmt.Errorf("rel path of file: %s", err)
			}
			if relPath == "." {
				relPath, err = filepath.Rel(rootPath, path)
				if err != nil {
					return fmt.Errorf("rel path for '.': %s", err)
				}
				workfiles = append(workfiles, relPath)
				return nil
			}
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
			relPath, err := filepath.Rel(rootPath, path)
			if err != nil {
				return err
			}
			workfiles = append(workfiles, relPath)
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	return workfiles, nil
}
