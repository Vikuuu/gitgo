package gitgo

import (
	"fmt"
	"math/rand"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func RemoveIgnoreFiles(input []os.DirEntry, ignore []string) []os.DirEntry {
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

func createTempFile(dirName, dbPath string) (*os.File, string, error) {
	tFileName := generateGitTempFileName(".temp-obj-")
	temp := filepath.Join(dbPath, dirName, tFileName)
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

func getUTCOffset(t time.Time) string {
	_, offset := t.Zone()

	offsetHour := offset / 3600
	offsetMin := (offset % 3600) / 60

	sign := "+"
	if offset < 0 {
		sign = "-"
	}

	return fmt.Sprintf("%s%02d%02d", sign, offsetHour, offsetMin)
}

func FirstLine(s string) string {
	firstLine, _, _ := strings.Cut(s, "\n")
	return firstLine
}
