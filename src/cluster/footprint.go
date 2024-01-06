package cluster

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/agaraleas/DecentralizedNetworkSync/logging"
	"github.com/agaraleas/DecentralizedNetworkSync/networking"
)

type Footprint struct {
	Port networking.Port `json:"port"`
	Hostname string		 `json:"hostname"`
	Pid int				 `json:"pid"`
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
		Port: networking.Port(port),
		Hostname: hostname,
		Pid: pid,
	}
	return &footprint, nil
}

func (f Footprint) String() string {
	return fmt.Sprintf("footprint_%s_%d_%d", f.Hostname, f.Port, f.Pid)
}

func (f Footprint) ToFile(folderPath string) error {
	_, err := os.Stat(folderPath)
	if os.IsNotExist(err) {
		logging.Log.Errorf("Path %s does not exist", folderPath)
		return err
	}
	
	fileName := f.String() + ".json"
	filePath := filepath.Join(folderPath, fileName)
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