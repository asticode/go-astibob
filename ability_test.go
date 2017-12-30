package astibob

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAbilityKey(t *testing.T) {
	assert.Equal(t, "test", abilityKey("Test"))
	assert.Equal(t, "test-1", abilityKey("Test 1"))
	assert.Equal(t, "t-est_1", abilityKey("T?est_1"))
}
