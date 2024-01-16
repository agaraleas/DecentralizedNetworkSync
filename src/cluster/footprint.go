package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

const FootprintFileNamePrefix = "footprint"

type Footprint struct {
	Port     networking.Port `json:"port"`
	Hostname string          `json:"hostname"`
	Pid      int             `json:"pid"`
}

func NewFootprint() (*Footprint, error) {
	port, err := networking.FindFreePort()
	if err != nil {
		logging.Log.Error("Could assign Port to Footprint")
		return nil, err
	}

	hostname, err := os.Hostname()
	if err != nil {
		logging.Log.Error("Could assign Hostname to Footprint")
		return nil, err
	}

	pid := os.Getpid()

	footprint := Footprint{
		Port:     networking.Port(port),
		Hostname: hostname,
		Pid:      pid,
	}
	return &footprint, nil
}

func (f Footprint) String() string {
	return fmt.Sprintf("%s_%s_%d_%d", FootprintFileNamePrefix, f.Hostname, f.Port, f.Pid)
}

func (f Footprint) ToFile(dirPath string) error {
	_, err := os.Stat(dirPath)
	if os.IsNotExist(err) {
		logging.Log.Errorf("Path %s does not exist", dirPath)
		return err
	}

	fileName := f.String() + ".json"
	filePath := filepath.Join(dirPath, fileName)
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(f)
}

func (f *Footprint) FromFile(filePath string) error {
	file, err := os.Open(filePath)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(f)
}

func ContainsFootprintFiles(dirPath string) (bool, error) {
	fileInfo, err := os.Stat(dirPath)
	if err != nil {
		logging.Log.Errorf("Failed to get info of directory: %s", dirPath)
		return false, err
	}

	if !fileInfo.IsDir() {
		logging.Log.Errorf("Path %s does not point to a directory", dirPath)
		return false, err
	}

	dirEntries, err := os.ReadDir(dirPath)
	if err != nil {
		logging.Log.Errorf("Failed to get entries of %s", dirPath)
		return false, err
	}

	for _, entry := range dirEntries {
		if !entry.IsDir() && IsFootprintFile(entry.Name()) {
			return true, nil
		}
	}

	return false, nil
}

func IsFootprintFile(fileName string) bool {
	return strings.HasPrefix(fileName, FootprintFileNamePrefix)
}
