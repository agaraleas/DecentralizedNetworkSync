package cluster

import (
	"encoding/json"
	"fmt"
	"os"
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

func (f Footprint) toFile() error {
	filename := f.String() + ".json"
	file, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	return encoder.Encode(f)
}

func (f *Footprint) fromFile(filename string) error {
	file, err := os.Open(filename)
	if err != nil {
		return err
	}
	defer file.Close()

	decoder := json.NewDecoder(file)
	return decoder.Decode(f)
}