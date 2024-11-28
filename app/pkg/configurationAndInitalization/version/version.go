package version

import (
	"embed"
	"encoding/json"
	"fmt"

	log "github.com/sirupsen/logrus"
)

// Embed variable of version.json
//
//go:embed version.json
var versionJSON embed.FS

type VersionJSON struct {
	Symantic string `json:"symantic" yaml:"symantic"`
	Hash     string `json:"hash" yaml:"hash"`
}

// Read reads the version file and returns the JSON information within the version.json file
func Read() (string, string) {
	// Unmarshal the version file embedded in the binary
	readVersionJSON, err := versionJSON.ReadFile("version.json")
	if err != nil {
		log.Error("Error reading version file", err)
	}
	var v VersionJSON
	err = json.Unmarshal(readVersionJSON, &v)
	if err != nil {
		log.Panic("Error unmarshalling version file", err)
	}
	return v.Symantic, v.Hash
}

func SymanticString() string {
	symantic, _ := Read()
	return fmt.Sprintf(symantic)
}

func HashString() string {
	_, hash := Read()
	return fmt.Sprintf(hash)
}

func ReadVersionJSON() string {
	readVersionJSON, err := versionJSON.ReadFile("version.json")
	if err != nil {
		log.Panic("Error reading version file", err)
	}
	return string(readVersionJSON)
}
