package cluster

import (
	"os"
	"testing"
	"reflect"

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

	err := expectedFootprint.toFile()
	require.Nil(t, err, "Failed to write Footprint to file")

	filename := expectedFootprint.String() + ".json"
	_, err = os.Stat(filename)
	require.False(t, os.IsNotExist(err), "File does not exist")

	var footprint Footprint
	err = footprint.fromFile(filename)
	assert.Nil(t, err, "Failed to read Footprint from file")

	assert.True(t, reflect.DeepEqual(footprint, expectedFootprint), "Footprints are not equal")

	err = os.Remove(filename)
	require.Nil(t, err, "Failed to delete Footprint file")
}