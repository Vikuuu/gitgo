package gitgo

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
)

func removeIgnoreFiles(input []os.DirEntry, ignore []string) []os.DirEntry {
	ignoreMap := make(map[string]bool)
	for _, v := range ignore {
		ignoreMap[v] = true
	}

	var result []os.DirEntry
	for _, v := range input {
		if !ignoreMap[v.Name()] {
			result = append(result, v)
		}
	}

	return result
}

func createTempFile(dirName string) (*os.File, string, error) {
	tFileName := generateGitTempFileName(".temp-obj-")
	temp := filepath.Join(DBPATH, dirName, tFileName)
	t, err := os.OpenFile(temp, os.O_RDWR|os.O_CREATE|os.O_EXCL, 0644)
	if err != nil {
		return nil, "", fmt.Errorf("Err creating temp file: %s", err)
	}

	return t, temp, nil
}

func generateGitTempFileName(prefix string) string {
	randomInt := (rand.Intn(999999) + 1)
	return prefix + strconv.Itoa(randomInt)
}
