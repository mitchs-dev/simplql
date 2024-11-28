package auth

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"fmt"

	"github.com/mitchs-dev/library-go/processor"
	log "github.com/sirupsen/logrus"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
)

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
