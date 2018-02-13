package configstore

import (
	"encoding/json"
	"io/ioutil"
	"os"

	"github.com/snikch/api/config"
)

var loc = config.String("CONFIGURATION_FILE", ".config.json")

// Load attempts to load the json configuration at the supplied location.
func Load(data interface{}) error {
	contents, err := ioutil.ReadFile(loc)
	if err != nil {
		// If the file doesn't exist, this isn't considered an error.
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	return json.Unmarshal(contents, data)
}

// Save will persist the supplied configuration struct to disk as json.
func Save(data interface{}) error {
	contents, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	return ioutil.WriteFile(loc, contents, 0644)
}
