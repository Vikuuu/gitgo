package datastr

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewSkipList(t *testing.T) {
	sl := NewSkipList()
	assert.NotNil(t, sl)
}
