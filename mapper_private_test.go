package cypherx

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsValidDest(t *testing.T) {
	s := struct{}{}
	assert.True(t, isValidDest(&s))
	assert.False(t, isValidDest(s))
	assert.False(t, isValidDest(map[string]interface{}{}))
}
