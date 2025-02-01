package utilites

import (
	"math/rand"
	"strconv"
)

func GenerateGitTempFileName(prefix string) string {
	randomInt := (rand.Intn(999999) + 1)
	return prefix + strconv.Itoa(randomInt)
}
