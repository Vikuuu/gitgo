package main

import (
	"encoding/hex"
	"io"
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
	stats map[string]os.FileInfo,
) error {
	fileStats, err := listDir(cmd.repo.Path, prefix)
	if err != nil {
		return err
	}
	for rel, stat := range fileStats {
		if index.IsTracked(rel) {
			if stat.IsDir() {
				if err := scanWorkspace(cmd, untracked, rel, index, stats); err != nil {
					return err
				}
			} else {
				stats[rel] = stat
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

func detectWorkspaceChanges(
	cmd command,
	changed datastr.SortedSet,
	index *gitgo.Index,
	stats map[string]os.FileInfo,
) {
	for name, entry := range index.IndexEntries() {
		checkIndexEntry(index, changed, stats, &entry, name, cmd.repo.Path)
	}
}

func checkIndexEntry(
	index *gitgo.Index,
	changed datastr.SortedSet,
	stats map[string]os.FileInfo,
	entry *gitgo.IndexEntry,
	name, path string,
) {
	// Check's if the file size has changed or something
	// noticable.
	stat := stats[name]
	if !entry.StatMatch(stat) {
		changed.Add(name)
		return
	}

	// Check with the time stamp
	if entry.TimeMatch(stat) {
		return
	}

	// If that is not the case we will check the content of
	// the file in question.
	f, err := os.OpenFile(filepath.Join(path, name), os.O_RDONLY, 0644)
	if err != nil {
		panic("err checkIndexEntry opening file: " + err.Error())
	}
	data, err := io.ReadAll(f)
	if err != nil {
		panic("err checkIndexEntry reading file: " + err.Error())
	}
	blob := gitgo.Blob{Data: data}.Init()
	oid := hex.EncodeToString(gitgo.GetHash(*blob))

	if oid == entry.Oid {
		// If the file content is same, but the file has different
		// metadata on the disk, update them so that we can
		// use them next time.
		index.UpdateEntryStat(entry, stat)
	} else {
		changed.Add(name)
	}
}
