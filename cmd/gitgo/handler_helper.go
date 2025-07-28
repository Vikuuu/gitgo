package main

import (
	"os"
	"path/filepath"

	"github.com/Vikuuu/gitgo"
	"github.com/Vikuuu/gitgo/internal/datastr"
)

func scanWorkspace(
	cmd command,
	untracked datastr.SortedSet,
	prefix string,
	index *gitgo.Index,
) error {
	stats, err := listDir(cmd.repo.Path, prefix)
	if err != nil {
		return err
	}
	for rel, stat := range stats {
		if index.IsTracked(rel) {
			if stat.IsDir() {
				if err := scanWorkspace(cmd, untracked, rel, index); err != nil {
					return err
				}
			}
		} else {
			trackablefile, err := trackableFile(rel, stat, index, cmd)
			if err != nil {
				return err
			}
			if trackablefile {
				if stat.IsDir() {
					rel += string(filepath.Separator)
				}
				untracked.Add(rel)
			}
		}
	}

	return nil
}

func listDir(base, dirname string) (map[string]os.FileInfo, error) {
	path := filepath.Join(base, dirname)

	dirEntries, err := os.ReadDir(path)
	if err != nil {
		return nil, err
	}
	stats := make(map[string]os.FileInfo, len(dirEntries))

	for _, dirEntry := range dirEntries {
		name := dirEntry.Name()
		if _, skip := gitgo.G_ignore[name]; skip {
			continue
		}

		rel, err := filepath.Rel(base, filepath.Join(path, name))
		if err != nil {
			return nil, err
		}

		info, err := dirEntry.Info()
		if err != nil {
			return nil, err
		}
		stats[rel] = info
	}

	return stats, nil
}

func trackableFile(path string, stat os.FileInfo, index *gitgo.Index, cmd command) (bool, error) {
	if stat == nil {
		return false, nil
	}

	if !stat.IsDir() {
		return !index.IsTracked(path), nil
	}

	items, err := listDir(cmd.repo.Path, path)
	if err != nil {
		return false, err
	}
	files := make(map[string]os.FileInfo)
	dirs := make(map[string]os.FileInfo)
	// dirs := []os.FileInfo{}

	for rel, item := range items {
		if item.IsDir() {
			dirs[rel] = item
			// dirs = append(dirs, item)
		} else {
			files[rel] = item
			// files = append(files, item)
		}
	}

	for filePath, file := range files {
		ok, err := trackableFile(filePath, file, index, cmd)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	for dirPath, dir := range dirs {
		ok, err := trackableFile(dirPath, dir, index, cmd)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}

	return false, nil
}
