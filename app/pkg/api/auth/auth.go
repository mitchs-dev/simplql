// This package differs from the api/auth package as this provides functions that are used in other packages not just the api package.
package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/mitchs-dev/library-go/encryption"
	"github.com/mitchs-dev/library-go/processor"
	"github.com/mitchs-dev/library-go/streaming"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	"gopkg.in/yaml.v2"

	jwtLib "github.com/mitchs-dev/library-go/jwt"
	log "github.com/sirupsen/logrus"
)

var c configuration.Configuration

// RunAuthChecks will check the authentication header and then check if the user exists via username and password or JWT and return a boolean, and the user's id
func RunAuthChecks(value, database, correlationID string, roleCheckList []string) (string, error) {

	c.GetConfig()
	log.Debug("Running authentication checks (C: " + correlationID + ")")

	// Get the name, password, and JWT from the authentication header
	name, password, jwt, err := AuthenticationHeaderData(value, correlationID)
	if err != nil {
		return "", fmt.Errorf("failed to get authentication header data: " + err.Error())
	}

	var (
		userExists bool
		userID     string
		roles      []string
	)

	// Check if the user exists via username and password
	if name != "" && password != "" {
		log.Debug("Checking if user exists via username and password (C: " + correlationID + ")")
		userExists, userID, roles, err = CheckBasic(name, password, database)
		if err != nil {
			return "", fmt.Errorf("failed to check if user exists via username and password: " + err.Error())
		}

		if !userExists {
			log.Debug("User does not exist via username and password (C: " + correlationID + ")")
			return "", errors.New(globals.ErrorAuthenticationUserNotFound)
		}
	} else if jwt != "" {
		log.Debug("Checking if user exists via JWT (C: " + correlationID + ")")
		userExists, userID, _, roles, err = CheckJWT(jwt, database)
		if err != nil {
			if strings.Contains(err.Error(), globals.ErrorAuthenticationJWTExpired) {
				log.Warn("JWT is expired for: " + userID + " (C: " + correlationID + ")")
				return "", errors.New(globals.ErrorAuthenticationJWTExpired)
			}
			return "", fmt.Errorf("failed to check if user exists via JWT: " + err.Error())
		}

		if !userExists {
			log.Debug("User does not exist via JWT (C: " + correlationID + ")")
			return "", errors.New(globals.ErrorAuthenticationUserNotFound)
		}
	}
	log.Debug("Found user: " + userID + " (C: " + correlationID + ")")

	// Check if the user has the required roles
	if len(roleCheckList) != 0 {
		log.Debug("Checking if user has the required roles (C: " + correlationID + ")")
		for _, userRole := range roles {
			log.Debug("Checking user role: " + userRole + " (C: " + correlationID + ")")
			if strings.Contains(strings.ToLower(userRole), strings.ToLower(globals.RolesSystemAdmin)) {
				log.Debug("User is a system admin and has all roles (C: " + correlationID + ")")
				log.Info("User authenticated: " + userID + " (C: " + correlationID + ")")
				return userID, nil
			}
			for _, checkRole := range roleCheckList {
				checkRole = strings.ToLower(globals.SystemRolePrefix + checkRole)
				log.Debug("Checking required role: " + checkRole + " (C: " + correlationID + ")")
				if checkRole == strings.ToLower(userRole) {
					log.Debug("User has the required role: " + checkRole + " (C: " + correlationID + ")")
					log.Info("User authenticated: " + userID + " (C: " + correlationID + ")")
					return userID, nil
				}
			}
		}
		log.Debug("User does not have any required roles (C: " + correlationID + ")")
		return userID, errors.New(globals.ErrorAuthenticationNoRoles)
	}

	log.Debug("Request does not have any roles to check (C: " + correlationID + ")")
	log.Info("User authenticated (C: " + correlationID + ")")
	return userID, nil

}

type AuthRequestBody globals.AuthRequestBody

// GetAuthRequest will unmarshal the request body as JSON or YAML and return the request body
func (authItem *AuthRequestBody) GetAuthRequest(authRequestBody []byte, correlationID string) *AuthRequestBody {
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

// Reads the value of the authentication header and returns name, password, and JWT
func AuthenticationHeaderData(value, correlationID string) (string, string, string, error) {
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
func CheckBasic(name, password, database string) (bool, string, []string, error) {
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

func CheckJWT(requestJWT, database string) (bool, string, string, []string, error) {
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
	isValidToken, error := jwtLib.ValidateToken(requestJWT, dbJWT, jwtSHA256, GetJWTSigningKey(database))
	if error != nil {
		if strings.Contains(err.Error(), globals.ErrorAuthenticationJWTExpiredFromJWTLib) {
			return false, "", "", nil, fmt.Errorf(globals.ErrorAuthenticationJWTExpired)
		}
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
func SetJWTIssuer(database string) string {

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
func GetJWTSigningKey(database string) string {
	encryptedData, err := encryption.Encrypt(streaming.Encode(database), globals.EncryptionKey, globals.EncryptionIV)
	if err != nil {
		log.Error("Failed to encrypt the database name: " + err.Error())
	}
	log.Debug("Using signing key: " + encryptedData)
	return encryptedData
}
