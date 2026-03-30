package runtime

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageResponsiveSizes_UsesValidInput(t *testing.T) {
	t.Parallel()

	const input = "(max-width: 768px) 100vw, 672px"
	assert.Equal(t, input, ImageResponsiveSizes(input, 1200))
}

func TestImageResponsiveSizes_InvalidInputFallsBackToWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "40px", ImageResponsiveSizes("invalid-size-value", 40))
}

func TestImageResponsiveSizes_Uses100vwWhenNoFallbackWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, "100vw", ImageResponsiveSizes("", 0))
}

func TestImageResponsiveTargetWidth_UsesSlotMaxFromSizes(t *testing.T) {
	t.Parallel()

	sizes := "(max-width: 768px) 100vw, 672px"
	assert.Equal(t, 768, ImageResponsiveTargetWidth(1200, sizes))
}

func TestImageResponsiveTargetWidth_DoesNotExceedIntrinsicWidth(t *testing.T) {
	t.Parallel()

	assert.Equal(t, 40, ImageResponsiveTargetWidth(40, "100vw"))
}
