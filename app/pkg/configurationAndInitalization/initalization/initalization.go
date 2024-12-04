package initalization

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"

	log "github.com/sirupsen/logrus"
	flag "github.com/spf13/pflag"

	"github.com/mitchs-dev/library-go/encryption"
	"github.com/mitchs-dev/library-go/generator"
	"github.com/mitchs-dev/library-go/loggingFormatter"
	"github.com/mitchs-dev/library-go/processor"
	"github.com/mitchs-dev/simplQL/pkg/api/requests"
)

var c configuration.Configuration

func Run() {
	var generateConfig bool
	// Configure logging
	log.SetFormatter(&loggingFormatter.JSONFormatter{
		Prefix:   "ssql-",
		Timezone: "Local",
	})
	log.SetOutput(os.Stdout)
	// Parse command line flags
	flag.StringVarP(&globals.ConfigFile, "config", "c", "", "Path to the configuration file")
	flag.BoolVarP(&generateConfig, "generate-config", "g", false, "Generate a default configuration file")
	flag.Parse()
	if generateConfig {
		configuration.GenerateDefaultConfig()
	}
	if globals.ConfigFile == "" {
		log.Error("No configuration file specified")
		flag.PrintDefaults()
		os.Exit(1)
	}
	if !processor.DirectoryOrFileExists(globals.ConfigFile) {
		log.Fatal("Configuration file does not exist: " + globals.ConfigFile)
	}
	c.GetConfig()
	if c.Logging.Debug {
		log.SetLevel(log.DebugLevel)
		log.Debug("Debug logging enabled - Use with caution")
		log.Warn("Debug logging contains information that may be sensitive - It's highly recommended to disable debug logging in production")
		log.Warn("Debug logging contains more than normal logging that could potentially be performance heavy")
	} else {
		log.SetLevel(log.InfoLevel)
	}

	if c.Storage.Encryption.Enabled {
		runEncryptionInit()
	} else {
		// We still need to create the storage directory
		if !processor.DirectoryOrFileExists(c.Storage.Path) {
			if !processor.CreateDirectory(c.Storage.Path) {
				log.Fatal("Failed to create storage directory: " + c.Storage.Path)
			}
		}
	}

	runSessionConfigInit()

	// Create databases
	err := sqlWrapper.CreateDatabases()
	if err != nil {
		log.Fatal("Error when creating databases: " + err.Error())
	}

	// Init requests
	requests.Startup()
}

func runSessionConfigInit() {
	c.GetConfig()
	globals.UseJWT = c.Session.JWT.Enabled
	log.Debug("JWT enabled: " + fmt.Sprint(globals.UseJWT))
	globals.JWTTimeoutPeriod = c.Session.JWT.Timeout
	log.Debug("JWT timeout period: " + globals.JWTTimeoutPeriod)

}

func runEncryptionInit() {
	log.Info("Encryption is enabled")
	encryptionEnvironmentVariable := os.Getenv(globals.EncryptionKeyEnvironmentVariable)
	if encryptionEnvironmentVariable == "" {
		log.Debug("Encryption key environment variable not set - checking configuration file")
		if c.Storage.Encryption.Key == "" {
			log.Info("Encryption key was not set in the configuration file - Checking for existing key")
			encryptionKeyDir := c.Storage.Encryption.Path + "/" + globals.EncryptionKeyFile
			if !processor.DirectoryOrFileExists(encryptionKeyDir) {
				if !processor.CreateDirectory(c.Storage.Encryption.Path) {
					log.Fatal("Failed to create encryption key directory: " + c.Storage.Encryption.Path)
				}
				// Check if there are any database files in the storage directory
				files, err := os.ReadDir(c.Storage.Encryption.Path)
				if err != nil {
					log.Debug("Storage directory does not exist - Creating")
					if !processor.CreateDirectory(c.Storage.Path) {
						log.Fatal("Failed to create storage directory: " + c.Storage.Path)
					}
				}
				for _, file := range files {
					if !file.IsDir() {
						if strings.HasSuffix(file.Name(), ".db") {
							log.Fatal("Database files found in storage directory, encryption is enabled and no key file exists - You should either remove the database files, set the encryption key, or set the encryption to false in the configuration file")
						}
					}
				}

				log.Info("Encryption key file does not exist - Generating new key")
				globals.EncryptionKey = encryption.GenerateKey()
				globals.EncryptionIV, err = encryption.GenerateIV()
				if err != nil {
					log.Fatal("Failed to generate IV: " + err.Error())
				}
				globals.EncryptionIVString = string(globals.EncryptionIV)
				encryptionKeyString := globals.EncryptionKey + ":" + string(globals.EncryptionIV)
				encryptedKey, err := encryption.Encrypt(encryptionKeyString, getIDCmdString(), globals.EncryptionIV)
				if err != nil {
					log.Fatal("Failed to encrypt the encryption key: " + err.Error())
				}
				if !processor.DirectoryOrFileExists(c.Storage.Path) {
					if !processor.CreateDirectory(c.Storage.Path) {
						log.Fatal("Failed to create encryption key directory: " + c.Storage.Path)
					}
				}
				if !processor.CreateFile(encryptionKeyDir, encryptedKey) {
					log.Fatal("Failed to create encryption key file: " + encryptionKeyDir)
				}
				log.Info("Successfully generated and stored in key file")
			} else {
				log.Info("Encryption key file exists - Reading key")
				rawEncryptionKey := string(processor.ReadFile(encryptionKeyDir))
				decryptedKey, err := encryption.Decrypt(rawEncryptionKey, getIDCmdString())
				if err != nil {
					log.Fatal("Failed to decrypt the encryption key: " + err.Error())
				}
				globals.EncryptionKey = string(strings.Split(decryptedKey, ":")[0])
				globals.EncryptionIV = []byte(strings.Split(decryptedKey, ":")[1])
				log.Debug("Encryption key set from key file")
			}

		} else {
			log.Info("Encryption key set from configuration file")
			if !strings.Contains(c.Storage.Encryption.Key, ":") {
				log.Fatal("Encryption key in configuration file is not in the correct format - Ensure that the key is in the format: key:iv (IV should be 16 bytes and a string)")
			}
			globals.EncryptionKey = strings.Split(c.Storage.Encryption.Key, ":")[0]
			globals.EncryptionIV = []byte(strings.Split(c.Storage.Encryption.Key, ":")[1])
		}
	} else {
		log.Info("Encryption key set from environment variable: " + globals.EncryptionKeyEnvironmentVariable)
		if !strings.Contains(globals.EncryptionKey, ":") {
			log.Fatal("Encryption key in environment variable is not in the correct format - Ensure that the key is in the format: key:iv (IV should be 16 bytes and a string)")
		}
		globals.EncryptionKey = strings.Split(encryptionEnvironmentVariable, ":")[0]
		globals.EncryptionIV = []byte(strings.Split(encryptionEnvironmentVariable, ":")[1])
	}

	// Encryption test
	log.Debug("Running encryption test")
	encryptionTestOriginalData := generator.RandomString(32)
	log.Debug("Original data: " + encryptionTestOriginalData)
	encryptedData, err := encryption.Encrypt(encryptionTestOriginalData, globals.EncryptionKey, globals.EncryptionIV)
	if err != nil {
		log.Fatal("Failed to encrypt data: " + err.Error())
	}
	log.Debug("Encrypted data: " + encryptedData)
	decryptedData, err := encryption.Decrypt(encryptedData, globals.EncryptionKey)
	if err != nil {
		log.Fatal("Failed to decrypt data: " + err.Error())
	}
	log.Debug("Decrypted data: " + decryptedData)
	if encryptionTestOriginalData != decryptedData {
		log.Fatal("Decrypted data does not match original data")
	}
	log.Debug("Encryption test passed")

	// Data processing test
	log.Debug("Running data processing test")
	dataProcessingTestOriginalData := generator.RandomString(32)
	dataProcessingTestEncryptData := data.Process(dataProcessingTestOriginalData).(string)
	if dataProcessingTestEncryptData == globals.ErrorDataProcessing {
		log.Fatal("Failed to process data - Failure point: Encrypt")
	}
	log.Debug("Data processing test - Encrypted data: " + dataProcessingTestEncryptData)
	dataProcessingTestDecryptData := data.Process(dataProcessingTestEncryptData).(string)
	if dataProcessingTestDecryptData == globals.ErrorDataProcessing {
		log.Fatal("Failed to process data - Failure point: Decrypt")
	}
	log.Debug("Data processing test - Decrypted data: " + dataProcessingTestDecryptData)
	if dataProcessingTestOriginalData != dataProcessingTestDecryptData {
		log.Fatal("Processed data does not match original data")
	}
	log.Debug("Data processing test passed")
}

// This function runs the ID command to get the ID of the user running the application
// This is used as a validation measure for the encryption key
func getIDCmdString() string {
	log.Debug("Getting user ID for encryption key")
	uid := os.Getuid()
	return strconv.Itoa(uid)
}
