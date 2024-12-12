// data manages the processing of data (Encrypt, Decrypt, etc)
package data

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/mitchs-dev/library-go/encryption"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	log "github.com/sirupsen/logrus"
)

var c configuration.Configuration

var isAWildCard bool

// Process processes the data prior to storage or retrieval
func Process(data interface{}) interface{} {

	c.GetConfig()

	if globals.IsTransactionExecution {
		log.Debug("Transaction execution - skipping data processing")
		return data

	}

	log.Debug("Original data: " + fmt.Sprintf("%v", data))

	if strings.HasPrefix(fmt.Sprintf("%v", data), "%") && strings.HasSuffix(fmt.Sprintf("%v", data), "%") {
		log.Debug("Data is a wildcard")
		isAWildCard = true
		data = strings.TrimPrefix(fmt.Sprintf("%v", data), "%")
		data = strings.TrimSuffix(fmt.Sprintf("%v", data), "%")
	}

	var originalFormat string
	// Check the type of the data
	switch data.(type) {
	case string:
		log.Debug("Data: " + data.(string))
		originalFormat = "string"
		_, err := json.Marshal(data)
		if err == nil {
			if strings.HasPrefix(data.(string), "[") && strings.HasSuffix(data.(string), "]") {
				originalFormat = "[]JSONstring"
				log.Debug("Data is a JSON array - setting as []string")
			}
		}
		// Check if the string is actually an int
		if _, err = strconv.Atoi(data.(string)); err == nil {
			originalFormat = "int"
			log.Debug("Data is an int - setting as int")
			break
		}
		// Check if the string is actually an int64
		if _, err = strconv.ParseInt(data.(string), 10, 64); err == nil {
			originalFormat = "int64"
			log.Debug("Data is an int64 - setting as int64")
			break
		}
		// Check if the string is actually a float64
		if _, err = strconv.ParseFloat(data.(string), 64); err == nil {
			originalFormat = "float64"
			log.Debug("Data is a float64 - setting as float64")
			break
		}
		// Check if the string is actually a bool
		if _, err = strconv.ParseBool(data.(string)); err == nil {
			originalFormat = "bool"
			log.Debug("Data is a bool - setting as bool")
			break
		}

	case []byte:
		originalFormat = "[]byte"
	case int:
		originalFormat = "int"
	case float64:
		originalFormat = "float64"
	case int64:
		originalFormat = "int64"
	case bool:
		originalFormat = "bool"
	case []string:
		originalFormat = "[]string"
		// Check if the array is actually []int
		_, err := strconv.Atoi(data.([]string)[0])
		if err == nil {
			originalFormat = "[]int"
			log.Debug("Data is an array of ints - setting as []int")
		}
		// Check if the array is actually []float64
		_, err = strconv.ParseFloat(data.([]string)[0], 64)
		if err == nil {
			originalFormat = "[]float64"
			log.Debug("Data is an array of float64s - setting as []float64")
		}
		// Check if the array is actually []int64
		_, err = strconv.ParseInt(data.([]string)[0], 10, 64)
		if err == nil {
			originalFormat = "[]int64"
			log.Debug("Data is an array of int64s - setting as []int64")
		}
		// Check if the array is actually []bool
		_, err = strconv.ParseBool(data.([]string)[0])
		if err == nil {
			originalFormat = "[]bool"
			log.Debug("Data is an array of bools - setting as []bool")
		}
	case []interface{}:
		log.Warn("Couldn't determine original format - assuming []string")
		originalFormat = "[]string"
	case *interface{}:
		originalFormat = "*interface{}"
	default:
		log.Error("Data type not supported: " + fmt.Sprintf("%T", data))
		return globals.ErrorDataProcessing
	}

	log.Debug("Original format: " + originalFormat)

	var dataAsString string
	// Format the data as a string
	switch originalFormat {
	case "string":
		dataAsString = data.(string)
	case "[]JSONstring":
		dataAsStringArray := strings.Split(data.(string), ",")
		for _, value := range dataAsStringArray {
			value = strings.TrimPrefix(value, "[")
			value = strings.TrimSuffix(value, "]")
			value = strings.TrimPrefix(value, "\"")
			value = strings.TrimSuffix(value, "\"")
			dataAsString += value + " "
		}
		dataAsString = strings.TrimSuffix(dataAsString, ",")
		originalFormat = "[]string"
	case "map[interface{}]interface{}":
		var result strings.Builder
		for key, value := range data.(map[interface{}]interface{}) {
			result.WriteString(fmt.Sprintf("%v: %v\n", key, value))
		}
		dataAsString = result.String()
	case "*interface{}":
		log.Debug("Data is *interface{} - processing")
		if dataPtr, ok := data.(*interface{}); ok {
			log.Debug("Data is *interface{} - checking if nil")
			if dataPtr != nil {
				log.Debug("Data is not nil - checking underlying value")
				value := reflect.ValueOf(*dataPtr)
				if value.IsValid() && !value.IsNil() {
					log.Debug("Underlying value is valid and not nil - setting as string")
					dataAsString = fmt.Sprintf("%v", value.Interface())
				} else {
					log.Debug("Underlying value is nil - setting as <nil>")
					dataAsString = "<nil>"
				}
			} else {
				log.Debug("Data is nil - setting as <nil>")
				dataAsString = "<nil>"
			}
		} else {
			log.Debug("Data is not *interface{} - setting as <invalid type>")
			dataAsString = "<invalid type>"
		}
	default:
		dataAsString = fmt.Sprintf("%s", data)
	}

	log.Debug("Data formatted as string: " + dataAsString)

	log.Debug("Processing data")

	// If encryption is enabled, de/encrypt the data
	if c.Storage.Encryption.Enabled {
		log.Debug("Encryption enabled - running through encryption process")
		// Try to decrypt the data first
		formattedData, formattedDataOriginalFormat := splitDataFromOriginalFormatHeader(dataAsString)
		decryptedData, dErr := Decrypt(formattedData)
		if dErr != nil {
			log.Debug("Could not decrypt data - trying to encrypt")
			// If decryption fails, try to encrypt the data
			encryptedData, eErr := Encrypt(dataAsString)
			if eErr != nil {
				log.Error("Could not de/encrypt data - Ensure that you are not trying to encrypt data when your database is not configured to use encryption")
				log.Error("Decryption error: " + dErr.Error())
				log.Error("Encryption error: " + eErr.Error())
				return globals.ErrorDataProcessing
			} else {
				log.Debug("Successfully encrypted data")
				encryptedDataFinal := setOriginalFormatHeader(encryptedData, originalFormat)
				if encryptedDataFinal == globals.ErrorDataProcessing {
					log.Error("Failed to set original format header")
					return globals.ErrorDataProcessing
				}
				log.Debug("Encrypted data: " + encryptedDataFinal)
				if isAWildCard {
					encryptedDataFinal = "%" + encryptedDataFinal + "%"
				}
				return encryptedDataFinal
			}
		} else {
			log.Debug("Successfully decrypted data")
			decryptedDataFinal, err := formatDataToOriginalFormat(decryptedData, formattedDataOriginalFormat)
			if err != nil {
				log.Error("Failed to convert decrypted data to original format: " + err.Error())
				return globals.ErrorDataProcessing
			}
			log.Debug("Returning data: " + fmt.Sprintf("%v", decryptedDataFinal))
			if isAWildCard {
				decryptedDataFinal = "%" + fmt.Sprintf("%v", decryptedDataFinal) + "%"
			}
			return decryptedDataFinal
		}
		// Otherwise, return the data as is
	} else {
		if strings.Contains(dataAsString, globals.EncryptionOriginalFormatHeaderStart) && strings.Contains(dataAsString, globals.EncryptionOriginalFormatHeaderEnd) {
			log.Fatal("Data appears to be encrypted, but encryption is currently disabled. Please enable encryption to access the data. Note: Disabling encryption after data has been encrypted will result in data loss.")
		}
		formatData, err := formatDataToOriginalFormat(dataAsString, originalFormat)
		if err != nil {
			log.Error("Failed to convert data to original format: " + err.Error())
			return globals.ErrorDataProcessing
		}
		log.Debug("Returning data: " + fmt.Sprintf("%v", formatData))
		return formatData
	}
}

// Sets the original format header for later decryption
func setOriginalFormatHeader(data string, originalFormat string) string {
	log.Debug("Setting original format header (" + originalFormat + ")")
	return strings.Replace(globals.EncryptionOriginalFormatHeader, globals.EncryptionOriginalFormatVar, originalFormat, -1) + data
}

// Splits the data from the original format header and also retrieves the original format type
func splitDataFromOriginalFormatHeader(data string) (string, string) {
	// Run validation
	if !strings.Contains(data, globals.EncryptionOriginalFormatHeaderStart) && !strings.Contains(data, globals.EncryptionOriginalFormatHeaderEnd) {
		log.Debug("Data does not contain original format header - Returning validation error")
		return globals.ErrorValidatingOriginalFormatHeader, globals.ErrorValidatingOriginalFormatHeader
	}
	// Split data from header
	encryptedData := strings.Split(data, globals.EncryptionOriginalFormatHeaderEnd)[1]
	formatHeader := strings.Split(data, globals.EncryptionOriginalFormatHeaderEnd)[0]
	originalFormat := strings.Replace(formatHeader, globals.EncryptionOriginalFormatHeaderStart, "", 1)
	originalFormat = strings.Replace(originalFormat, globals.EncryptionOriginalFormatHeaderEnd, "", 1)
	return encryptedData, originalFormat
}

// Returns data from string to its original format
func formatDataToOriginalFormat(data, originalFormat string) (interface{}, error) {
	log.Debug("Converting data to original format: " + originalFormat)
	var convertedData interface{}
	var err error
	switch originalFormat {
	case "string":
		convertedData = data
	case "[]byte":
		convertedData = []byte(data)
	case "int":
		convertedData, err = strconv.Atoi(data)
		if err != nil {
			return nil, err
		}
	case "float64":
		data = strings.TrimPrefix(data, "%!s(float64=")
		data = strings.TrimSuffix(data, ")")
		if !strings.Contains(data, ".") {
			log.Debug("Data was detected as float64, but is actually an int - converting to int")
			convertedData, err = strconv.Atoi(data)
			if err != nil {
				return nil, err
			}
		} else {
			convertedData, err = strconv.ParseFloat(data, 64)
			if err != nil {
				return nil, err
			}
		}
	case "int64":
		data = strings.TrimPrefix(data, "%!s(int64=")
		data = strings.TrimSuffix(data, ")")
		convertedData, err = strconv.ParseInt(data, 10, 64)
		if err != nil {
			return nil, err
		}
	case "bool":
		data = strings.TrimPrefix(data, "%!s(bool=")
		data = strings.TrimSuffix(data, ")")
		convertedData, err = strconv.ParseBool(data)
		if err != nil {
			return nil, err
		}
	case "[]string":
		log.Debug("Datatype is []string - Storing as CSV")
		convertedData = strings.ReplaceAll(data, " ", ",")
		convertedData = strings.TrimSuffix(convertedData.(string), "]")
		convertedData = strings.TrimPrefix(convertedData.(string), "[")
		convertedData = strings.TrimSuffix(convertedData.(string), ",")
		convertedData = strings.Split(convertedData.(string), ",")
	case "*interface{}":
		log.Debug("Format is *interface{} - Returning as is")
		var newInterface interface{} = data
		convertedData = &newInterface
	default:
		return nil, fmt.Errorf("original format not supported")
	}
	return convertedData, nil
}

// Encrypt encrypts the data
func Encrypt(data string) (string, error) {
	return encryption.Encrypt(data, globals.EncryptionKey, globals.EncryptionIV)
}

// Decrypt decrypts the data
func Decrypt(data string) (string, error) {
	return encryption.Decrypt(data, globals.EncryptionKey)
}
