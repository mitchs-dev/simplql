package configuration

import (
	"embed"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"

	libGoConfiguration "github.com/mitchs-dev/library-go/configuration"
	"github.com/mitchs-dev/library-go/processor"
	"gopkg.in/yaml.v2"
)

//go:embed default.yaml
var defaultConfig embed.FS

var configFile string

// Configuration is a struct that holds the configuration
type Configuration struct {
	Logging struct {
		Debug        bool `json:"debug" yaml:"debug"`
		Transactions struct {
			Enabled          bool `json:"enabled" yaml:"enabled"`
			LogSelectQueries bool `json:"logSelectQueries" yaml:"logSelectQueries"`
		}
	} `json:"logging" yaml:"logging"`
	Network struct {
		Port          int    `json:"port" yaml:"port"`
		ListenAddress string `json:"listenAddress" yaml:"listenAddress"`
		TLS           struct {
			Enabled bool   `json:"enabled" yaml:"enabled"`
			Cert    string `json:"certFile" yaml:"cert"`
			Key     string `json:"keyFile" yaml:"key"`
		} `json:"tls" yaml:"tls"`
	} `json:"network" yaml:"network"`
	Session struct {
		JWT struct {
			Enabled bool   `json:"enabled" yaml:"enabled"`
			Timeout string `json:"timeout" yaml:"timeout"`
		} `json:"jwt" yaml:"jwt"`
		Default struct {
			Name     string `json:"name" yaml:"name"`
			Password string `json:"password" yaml:"password"`
		} `json:"default" yaml:"default"`
	} `json:"session" yaml:"session"`
	Storage struct {
		Encryption struct {
			Enabled bool   `json:"enabled" yaml:"enabled"`
			Path    string `json:"path" yaml:"path"`
			Key     string `json:"key" yaml:"key"`
		} `json:"encryption" yaml:"encryption"`
		Path string `json:"path" yaml:"path"`
	} `json:"storage" yaml:"storage"`
	Databases []ConfigurationDatabaseEntry `json:"databases" yaml:"databases"`
}

// ConfigurationDatabaseEntry is a struct that holds the configuration for a database
type ConfigurationDatabaseEntry struct {
	Name    string                                  `json:"name" yaml:"name"`
	Version int                                     `json:"version" yaml:"version"`
	Tables  []ConfigurationDatabaseEntryTablesEntry `json:"tables" yaml:"tables"`
}

// ConfigurationDatabaseEntryTablesEntry is a struct that holds the configuration for a database table
type ConfigurationDatabaseEntryTablesEntry struct {
	Name    string                                   `json:"name" yaml:"name"`
	Columns []ConfigurationDatabaseEntryColumnsEntry `json:"columns" yaml:"columns"`
}

// ConfigurationDatabaseEntryColumnsEntry is a struct that holds the configuration for a database column
type ConfigurationDatabaseEntryColumnsEntry struct {
	Name       string `json:"name" yaml:"name"`
	Type       string `json:"type" yaml:"type"`
	PrimaryKey bool   `json:"primaryKey" yaml:"primaryKey"`
}

func (configItem *Configuration) GetConfig() *Configuration {
	// Read default configuration
	defaultConfigData, err := defaultConfig.ReadFile(globals.DefaultConfigFileName)
	if err != nil {
		log.Fatal("Could not read default configuration file: " + err.Error())
	}
	defaultConfigMap := make(map[interface{}]interface{})
	err = yaml.Unmarshal(defaultConfigData, &defaultConfigMap)
	if err != nil {
		log.Fatal("Could not parse default configuration file: " + err.Error())
	}
	// Read configuration file
	configFileData := processor.ReadFile(globals.ConfigFile)
	userConfigMap := make(map[interface{}]interface{})
	// Determine if file is JSON or YAML
	err = json.Unmarshal(configFileData, &userConfigMap)
	if err != nil {
		// Try YAML
		err := yaml.Unmarshal(configFileData, &userConfigMap)
		if err != nil {
			log.Fatal("Could not parse configuration file: " + err.Error())
		}
	}
	// Merge user configuration with default configuration
	mergedConfigMap := libGoConfiguration.MergeWithDefault(defaultConfigMap, userConfigMap)
	mergedConfigData, err := yaml.Marshal(mergedConfigMap)
	if err != nil {
		log.Fatal("Could not merge configuration files: " + err.Error())
	}
	err = yaml.Unmarshal(mergedConfigData, &configItem)
	if err != nil {
		log.Fatal("Could not parse merged configuration file: " + err.Error())
	}
	// Return configuration
	return configItem
}

// GenerateDefaultConfig generates a default configuration file
func GenerateDefaultConfig() {
	defaultConfigData, err := defaultConfig.ReadFile(globals.DefaultConfigFileName)
	if err != nil {
		log.Fatal("Could not read default configuration file: " + err.Error())
	}
	fmt.Println(string(defaultConfigData))
	os.Exit(0)
}
