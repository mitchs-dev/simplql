package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchs-dev/library-go/encryption"
	jwtLib "github.com/mitchs-dev/library-go/jwt"
	"github.com/mitchs-dev/library-go/processor"
	"github.com/mitchs-dev/library-go/streaming"
	log "github.com/sirupsen/logrus"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	"gopkg.in/yaml.v2"
)

type authRequestBody globals.AuthRequestBody

func (authItem *authRequestBody) getAuthRequest(authRequestBody []byte, correlationID string) *authRequestBody {
	if len(authRequestBody) == 0 {
		log.Error("Request body is empty (C: " + correlationID + ")")
		return nil
	}
	// Check if the request body is formatted as JSON
	err := json.Unmarshal(authRequestBody, authItem)
	// Check if the request body is formatted as JSON
	if err != nil {
		if strings.Contains(err.Error(), "invalid character") {
			// Next try to unmarshal as YAML
			err = yaml.Unmarshal(authRequestBody, authItem)
			if err != nil {
				log.Error("Failed to unmarshal request body as JSON or YAML: " + err.Error() + " (C: " + correlationID + ")")
				return nil
			}
		} else {
			log.Error("Failed to unmarshal request body as JSON: " + err.Error() + " (C: " + correlationID + ")")
			return nil
		}
	}
	return authItem
}

func validateSystemRoles(rolesAsInterface interface{}) bool {
	log.Debug("Role data: ", rolesAsInterface)
	log.Debug("Type of rolesAsInterface: ", fmt.Sprintf("%T", rolesAsInterface))

	rolesInterfaceSlice, ok := rolesAsInterface.([]interface{})
	if !ok {
		log.Error("Failed to convert roles to []interface{}")
		return false
	}

	var roles []string
	for _, role := range rolesInterfaceSlice {
		roleStr, ok := role.(string)
		if !ok {
			log.Error("Failed to convert role to string")
			return false
		}
		roles = append(roles, roleStr)
	}

	for _, role := range roles {
		if role != globals.RolesSystemAdmin && role != globals.RolesSystemUser && role != globals.RolesSystemReadOnly {
			log.Debug("Role: " + role + " is not a valid system role")
			return false
		} else {
			log.Debug("Role: " + role + " is a valid system role")
			return true
		}
	}
	log.Error("No roles seem to be specified for system role validation")
	return false
}

// Reads the value of the authentication header and returns name, password, and JWT
func authenticationHeaderData(value, correlationID string) (string, string, string, error) {
	c.GetConfig()
	if value == "" {
		return "", "", "", fmt.Errorf("authentication header is empty")
	}
	var (
		decodedValue string
		err          error
		name         string
		password     string
		jwt          string
	)
	log.Debug("Raw value: ", value)
	if strings.HasPrefix(strings.ToLower(value), strings.ToLower(globals.AuthenticationAuthorizationHeaderBearerPrefix)) {
		if !c.Session.JWT.Enabled {
			return "", "", "", fmt.Errorf(globals.ErrorJWTDisabled)
		}
		value = strings.TrimPrefix(value, globals.AuthenticationAuthorizationHeaderBearerPrefix)
		decodedValue, err = streaming.Decode(value)
		if err != nil {
			return "", "", "", fmt.Errorf("Failed to decode authentication header: " + err.Error())
		}
		log.Debug("Decoded value: ", decodedValue)
		// Check if the decoded value is a JWT
		log.Debug("Authentication header value is a JWT (C: " + correlationID + ")")
		log.Debug("Encoding JWT for validation (C: " + correlationID + ")")
		jwt = streaming.Encode(decodedValue)
	} else if strings.HasPrefix(strings.ToLower(value), strings.ToLower(globals.AuthenticationAuthorizationHeaderBasicPrefix)) {
		value = strings.TrimPrefix(value, globals.AuthenticationAuthorizationHeaderBasicPrefix)
		decodedValue, err = streaming.Decode(value)
		if err != nil {
			return "", "", "", fmt.Errorf("Failed to decode authentication header: " + err.Error())
		}
		log.Debug("Decoded value: ", decodedValue)
		// Split the decoded value into name and password
		name = strings.Split(decodedValue, ":")[0]
		password = strings.Split(decodedValue, ":")[1]
		log.Debug("Authentication header value is a name and password (C: " + correlationID + ")")
	} else {
		return "", "", "", fmt.Errorf("authentication header value is not in the correct format")
	}
	log.Debugf("Returning name (%s) password (%s) and JWT (%s)", name, password, jwt)
	return name, password, jwt, nil

}

// Check if the user exists via username and password and returns boolean, id, and roles
func checkBasic(name, password, database string) (bool, string, []string, error) {
	c.GetConfig()
	log.Debug("Checking if user (" + name + ") exists via username and password")

	// Check if the database exists
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	log.Debug("Checking database file path: " + dbFilePath)
	if !processor.DirectoryOrFileExists(dbFilePath) {
		return false, "", nil, fmt.Errorf("Database (" + database + ") does not exist")
	}
	// Create a wrapper for the SQL query
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		return false, "", nil, fmt.Errorf("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	// Check if the user exists
	query := "SELECT " + globals.UserEntryIDColumnName + "," + globals.UserRolesColumnName + " FROM " + globals.UsersTable + " WHERE " + globals.UserNameColumnName + " = ? AND " + globals.UserPasswordColumnName + " = ?"
	rows, err := wrapper.Query(query, name, password)
	if err != nil {
		return false, "", nil, fmt.Errorf("Failed to execute select query: " + err.Error())
	}
	defer rows.Close()
	var rolesAsString string
	var id string

	for rows.Next() {
		if err := rows.Scan(&id, &rolesAsString); err != nil {
			return false, "", nil, fmt.Errorf("Failed to scan row: " + err.Error())
		}
		id = data.Process(id).(string)
		log.Debug("User exists: " + name + "(" + id + ")")
		roles := data.Process(rolesAsString).([]string)
		if len(roles) != 0 {
			return true, id, roles, nil
		}
	}
	log.Debug("User does not exist: " + name)

	roles := []string{}
	return false, id, roles, nil
}

func checkJWT(requestJWT, database string) (bool, string, string, []string, error) {
	c.GetConfig()
	log.Debug("Checking if user exists via JWT")

	// Get the user ID from the token
	id, err := jwtLib.GetAudience(requestJWT)
	if err != nil {
		return false, "", "", nil, fmt.Errorf("failed to get audience from token: " + err.Error())
	}

	// Check if the database exists
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	log.Debug("Checking database file path: " + dbFilePath)
	if !processor.DirectoryOrFileExists(dbFilePath) {
		return false, "", "", nil, fmt.Errorf("Database (" + database + ") does not exist")
	}

	// Create a wrapper for the SQL query
	log.Debug("Ensuring the token exists in the database")
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		return false, "", "", nil, fmt.Errorf("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	// Check if the user exists
	query := "SELECT token,sha256 FROM " + globals.JWTTable + " WHERE token = ? AND " + globals.UserEntryIDColumnName + " = ?"
	rows, err := wrapper.Query(query, requestJWT, id)
	if err != nil {
		return false, "", "", nil, fmt.Errorf("Failed to execute select query: " + err.Error())
	}
	var dbJWT string
	var jwtSHA256 string
	log.Debug("Checking if user exists via JWT")
	if rows == nil {
		log.Debug("No rows found in JWT table")
		roles := []string{}
		return false, id, "", roles, nil
	}
	defer rows.Close()

	for rows.Next() {
		if err := rows.Scan(&dbJWT, &jwtSHA256); err != nil {
			return false, "", "", nil, fmt.Errorf("Failed to scan row: " + err.Error())
		}
		dbJWT = data.Process(dbJWT).(string)
		jwtSHA256 = data.Process(jwtSHA256).(string)
	}

	if dbJWT == "" {
		roles := []string{}
		return false, id, "", roles, errors.New(globals.ErrorNotExist)
	}

	if jwtSHA256 == "" {
		roles := []string{}
		return false, id, "", roles, errors.New(globals.ErrorNotExist)
	}

	log.Debug("JWT SHA256: " + jwtSHA256)
	isValidToken, error := jwtLib.ValidateToken(requestJWT, dbJWT, jwtSHA256, getJWTSigningKey(database))
	if error != nil {
		return false, "", "", nil, fmt.Errorf("Failed to validate token: " + error.Error())
	}
	if !isValidToken {
		return false, "", "", nil, fmt.Errorf("token is not valid")
	}

	log.Debug("User exists via JWT: " + id)

	// Get the roles for the user
	query = "SELECT " + globals.UserNameColumnName + "," + globals.UserRolesColumnName + " FROM " + globals.UsersTable + " WHERE " + globals.UserEntryIDColumnName + " = ?"
	rows, err = wrapper.Query(query, id)
	if err != nil {
		return false, "", "", nil, fmt.Errorf("Failed to execute select query: " + err.Error())
	}
	defer rows.Close()
	var name string
	var rolesAsString string
	for rows.Next() {
		if err := rows.Scan(&name, &rolesAsString); err != nil {
			return false, "", "", nil, fmt.Errorf("Failed to scan row: " + err.Error())
		}
		name = data.Process(name).(string)
		roles := data.Process(rolesAsString).([]string)
		if len(roles) != 0 {
			return true, id, name, roles, nil
		}
	}
	log.Debug("User name or roles do not exist: " + id)
	return false, id, "", nil, nil
}

// This sets the issuer for the JWT
func setJWTIssuer(database string) string {

	c.GetConfig()
	log.Debug("Generating SimplQL ID for database: " + database)

	// Check if the database exists
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	log.Debug("Checking database file path: " + dbFilePath)
	if !processor.DirectoryOrFileExists(dbFilePath) {
		return "NULL"
	}
	// Create a wrapper for the SQL query
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		return "NULL"
	}
	defer wrapper.Close()

	// Check if the user exists
	query := "SELECT version FROM " + globals.MetadataTable
	log.Debug("JWT Issuer select query: " + query)
	rows, err := wrapper.Query(query)
	if err != nil {
		return "NULL"
	}
	defer rows.Close()
	var version string
	for rows.Next() {
		if err := rows.Scan(&version); err != nil {
			log.Error("Failed to scan row: " + err.Error())
			return "NULL"
		}
	}
	return strings.ReplaceAll(globals.JWTIssuer, globals.SimplQLIdPlaceholder, database+"@v"+data.Process(version).(string))
}

// Get the signing key for the JWT (encrypted value for the database name)
func getJWTSigningKey(database string) string {
	encryptedData, err := encryption.Encrypt(streaming.Encode(database), globals.EncryptionKey, globals.EncryptionIV)
	if err != nil {
		log.Error("Failed to encrypt the database name: " + err.Error())
	}
	log.Debug("Using signing key: " + encryptedData)
	return encryptedData
}

func commitJWT(id, jwt string, timeout int64, database string) error {
	log.Debug("Committing JWT for user: " + id)
	c.GetConfig()
	// Generate SHA256 hash
	hash := sha256.Sum256([]byte(jwt))
	jwtSHA256 := "0x" + hex.EncodeToString(hash[:]) // Check if the database exists
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	log.Debug("Checking database file path: " + dbFilePath)
	if !processor.DirectoryOrFileExists(dbFilePath) {
		return fmt.Errorf("Database (" + database + ") does not exist")
	}
	// Create a wrapper for the SQL query
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		return fmt.Errorf("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	// Check if the ID exists in the JWT table
	selectQuery := "SELECT id FROM " + globals.JWTTable + " WHERE id = ?"
	selectArgs := []interface{}{id}
	log.Debug("JWT select query: " + selectQuery)
	log.Debug("JWT select args: ", selectArgs)
	var existingID string
	err = wrapper.QueryRow(selectQuery, selectArgs...).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("Failed to execute select query: " + err.Error())
	}

	//If the ID exists, perform an UPDATE
	if existingID != "" {
		updateQuery := "UPDATE " + globals.JWTTable + " SET token = ?, sha256 = ?, expiration = ? WHERE id = ?"
		updateArgs := []interface{}{jwt, jwtSHA256, timeout, id}
		log.Debug("JWT update query: " + updateQuery)
		log.Debug("JWT update args: ", updateArgs)
		_, err = wrapper.Execute(updateQuery, updateArgs...)
		if err != nil {
			return fmt.Errorf("Failed to execute update query: " + err.Error())
		}
	} else {
		// If the ID does not exist, perform an INSERT
		insertQuery := "INSERT INTO " + globals.JWTTable + " (id, token,sha256, expiration) VALUES (?,?, ?, ?)"
		insertArgs := []interface{}{id, jwt, jwtSHA256, timeout}
		log.Debug("JWT insert query: " + insertQuery)
		log.Debug("JWT insert args: ", insertArgs)
		_, err = wrapper.Execute(insertQuery, insertArgs...)
		if err != nil {
			return fmt.Errorf("Failed to execute insert query: " + err.Error())
		}
	}
	log.Debug("JWT committed for user: " + id)
	return nil

}

// Removes the JWT from the database hence removing the session
func deleteJWT(id, database string) error {
	log.Debug("Deleting JWT for user: " + id)
	c.GetConfig()
	// Check if the database exists
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	log.Debug("Checking database file path: " + dbFilePath)
	if !processor.DirectoryOrFileExists(dbFilePath) {
		return fmt.Errorf("Database (" + database + ") does not exist")
	}
	// Create a wrapper for the SQL query
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		return fmt.Errorf("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	// Check if the ID exists in the JWT table
	selectQuery := "SELECT id FROM " + globals.JWTTable + " WHERE id = ?"
	selectArgs := []interface{}{id}
	log.Debug("JWT select query: " + selectQuery)
	log.Debug("JWT select args: ", selectArgs)
	var existingID string
	err = wrapper.QueryRow(selectQuery, selectArgs...).Scan(&existingID)
	if err != nil && err != sql.ErrNoRows {
		return fmt.Errorf("Failed to execute select query: " + err.Error())
	}

	// If the ID exists, perform a DELETE
	if existingID != "" {
		log.Debug("Found existing ID (" + existingID + ") in the JWT table")
		deleteQuery := "DELETE FROM " + globals.JWTTable + " WHERE id = ?"
		deleteArgs := []interface{}{existingID}
		log.Debug("JWT delete query: " + deleteQuery)
		log.Debug("JWT delete args: ", deleteArgs)
		_, err = wrapper.Execute(deleteQuery, deleteArgs...)
		if err != nil {
			return fmt.Errorf("Failed to execute delete query: " + err.Error())
		}
	} else {
		return fmt.Errorf(globals.ErrorNotExist)
	}
	log.Debug("JWT deleted for user: " + id)
	return nil
}
