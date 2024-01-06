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
		Port: 8080,
		Hostname: "testhost",
		Pid: 1234,
	}

	expectedString := "footprint_testhost_8080_1234"
	assert.Equal(t, footprint.String(), expectedString, "Footprint string is not correct")
}

func TestFootprintFile(t *testing.T) {
	expectedFootprint := Footprint{
		Port: 8080,
		Hostname: "testhost",
		Pid: 1234,
	}

	folderPath := os.TempDir()
	err := expectedFootprint.ToFile(folderPath)
	require.Nil(t, err, "Failed to write Footprint to file")

	fileName := expectedFootprint.String() + ".json"
	filePath := filepath.Join(folderPath, fileName)

	var footprint Footprint
	err = footprint.FromFile(filePath)
	assert.Nil(t, err, "Failed to read Footprint from file")

	assert.True(t, reflect.DeepEqual(footprint, expectedFootprint), "Footprints are not equal")

	err = os.Remove(filePath)
	require.Nil(t, err, "Failed to delete Footprint file")
}