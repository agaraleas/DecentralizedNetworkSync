package cluster

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFootptintString(t *testing.T) {
	footprint := Footprint{
		Port:     8080,
		Hostname: "testhost",
		Pid:      1234,
	}

	expectedString := "footprint_testhost_8080_1234"
	assert.Equal(t, footprint.String(), expectedString, "Footprint string is not correct")
}

func TestFootprintFile(t *testing.T) {
	expectedFootprint := Footprint{
		Port:     8080,
		Hostname: "testhost",
		Pid:      1234,
	}

	tempDirPath := os.TempDir()
	err := expectedFootprint.ToFile(tempDirPath)
	require.Nil(t, err, "Failed to write Footprint to file")

	fileName := expectedFootprint.String() + ".json"
	filePath := filepath.Join(tempDirPath, fileName)

	defer func(filePath string) {
		err = os.Remove(filePath)
		require.Nil(t, err, "Failed to delete Footprint file")
	}(filePath)

	var footprint Footprint
	err = footprint.FromFile(filePath)
	assert.Nil(t, err, "Failed to read Footprint from file")

	assert.True(t, reflect.DeepEqual(footprint, expectedFootprint), "Footprints are not equal")
}

func TestContainsFootprintFiles(t *testing.T) {
	tempDirPath := os.TempDir()

	// Test case 1: No footprint files
	result, err := ContainsFootprintFiles(tempDirPath)
	assert.Nil(t, err, "Failed to check directory's contents")
	assert.False(t, result, "Expected false when no footprint files exist")

	// Test case 2: Footprint file present
	footprint := Footprint{
		Port:     8080,
		Hostname: "testhost",
		Pid:      1234,
	}
	footprint.ToFile(tempDirPath)

	fileName := footprint.String() + ".json"
	filePath := filepath.Join(tempDirPath, fileName)

	defer func(filePath string) {
		err = os.Remove(filePath)
		require.Nil(t, err, "Failed to delete Footprint file")
	}(filePath)

	result, err = ContainsFootprintFiles(tempDirPath)
	assert.Nil(t, err, "Failed to check directory's contents")
	assert.True(t, result, "Expected true when footprint files exist")

	// Test case 3: Invalid directory path
	result, err = ContainsFootprintFiles("invalid/path")
	assert.NotNil(t, err, "Expected error for invalid directory path")
	assert.False(t, result)

	// Test case 4: Non directory path
	result, err = ContainsFootprintFiles("invalid/path")
	assert.NotNil(t, err, "Expected error for non directory path")
	assert.False(t, result)
}

func TestIsFootprintFile(t *testing.T) {
	// Test case 1: Valid footprint file name
	result := IsFootprintFile("footprint_example.json")
	assert.True(t, result, "Expected true for a valid footprint file name")

	// Test case 2: Non-footprint file name
	result = IsFootprintFile("example.json")
	assert.False(t, result, "Expected false for a non-footprint file name")

	// Test case 3: Empty file name
	result = IsFootprintFile("")
	assert.False(t, result, "Expected false for an empty file name")

	// Additional test cases can be added as needed
}
