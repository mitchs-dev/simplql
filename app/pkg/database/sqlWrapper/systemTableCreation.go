package sqlWrapper

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"

	"github.com/mitchs-dev/library-go/generator"
	"github.com/mitchs-dev/library-go/processor"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	log "github.com/sirupsen/logrus"
)

func createTransactionsTable(database string) error {
	if !c.Logging.Transactions.Enabled {
		globals.IsTransactionExecution = false
		log.Debug("Transactions are disabled - Skipping transaction table creation")
	} else {
		dbFilePath := c.Storage.Path + "/" + database + ".db"
		wrapper, err := NewSQLiteWrapper(dbFilePath)
		if err != nil {
			log.Error("Error when creating transactions table: " + err.Error())
			return errors.New(globals.ErrorDatabaseInitialization)
		}
		defer wrapper.Close()
		// Create the transaction table
		createTransactionTableQuery := `CREATE TABLE IF NOT EXISTS ` + globals.TransactionsTable + ` (id INTEGER PRIMARY KEY AUTOINCREMENT,Timestamp DATETIME DEFAULT CURRENT_TIMESTAMP,userID INTEGER NOT NULL,actionType TEXT NOT NULL,affectedTable TEXT NOT NULL,recordID INTEGER NOT NULL,oldValues TEXT,newValues TEXT,ipAddress TEXT,status TEXT NOT NULL,errorMessage TEXT)`
		log.Debug("Creating transactions table with query: " + createTransactionTableQuery + " for database: " + database)
		_, err = wrapper.Execute(createTransactionTableQuery, globals.SystemUserID)
		if err != nil {
			log.Error("Error when creating transactions table: " + err.Error())
			return errors.New(globals.ErrorDatabaseInitialization)
		}
		log.Info("Successfully created system transactions table")
	}
	return nil
}

func createMetadataTable(database string, databaseVersion int) error {
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	wrapper, err := NewSQLiteWrapper(c.Storage.Path + "/" + database + ".db")
	if err != nil {
		log.Error("Error when creating metadata table: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	query := `CREATE TABLE IF NOT EXISTS ` + globals.MetadataTable + ` (version TEXT)`
	// Create metadata table
	_, err = wrapper.Execute(query, globals.SystemUserID)
	if err != nil {
		log.Error("Error when creating metadata table: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	query = `INSERT INTO ` + globals.MetadataTable + ` (version) VALUES (?)`
	args := fmt.Sprint(databaseVersion)
	_, err = wrapper.Execute(query, globals.SystemUserID, args)
	if err != nil {
		log.Error("Error when inserting database version: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	log.Info("Successfully created system metadata table")
	log.Info("Set database version to: " + fmt.Sprint(databaseVersion) + " for: " + database + " in metadata table")
	return nil
}

func createUsersTable(database string) error {
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	wrapper, err := NewSQLiteWrapper(c.Storage.Path + "/" + database + ".db")
	if err != nil {
		log.Error("Error when creating users table: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	query := `CREATE TABLE IF NOT EXISTS ` + globals.UsersTable + ` (id TEXT PRIMARY KEY, name TEXT, password TEXT, roles TEXT)`
	// Create users table
	_, err = wrapper.Execute(query, globals.SystemUserID)
	if err != nil {
		log.Error("Error when creating users table: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	log.Debug("Successfully created system users table")
	var userPassword string
	var randomPassword bool
	if os.Getenv(globals.SessionDefaultPasswordEnvironmentVariable) != "" {
		userPassword = os.Getenv(globals.SessionDefaultPasswordEnvironmentVariable)
		randomPassword = false
	} else if c.Session.Default.Password != "" {
		userPassword = c.Session.Default.Password
		randomPassword = false
	} else {
		userPassword = generator.RandomString(globals.UserPasswordLength)
		randomPassword = true
	}
	var userName string
	if os.Getenv(globals.SessionDefaultUsernameEnvironmentVariable) != "" {
		userName = os.Getenv(globals.SessionDefaultUsernameEnvironmentVariable)
	} else {
		userName = c.Session.Default.Name
	}
	userID := generator.RandomString(globals.UserIDLength)
	defaultRolesAsJSON, err := json.Marshal(globals.DefaultRoles)
	if err != nil {
		log.Error("Error when marshalling default roles: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	query = `INSERT INTO ` + globals.UsersTable + ` (id,name, password, roles) VALUES (?,?, ?, ?)`
	args := []interface{}{userID, userName, userPassword, string(defaultRolesAsJSON)}
	_, err = wrapper.Execute(query, globals.SystemUserID, args)
	if err != nil {
		log.Error("Error when inserting default user: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	if randomPassword {
		log.Info("Generated default user: " + userName + " with password: " + userPassword + " for database: " + database)
		log.Warn("Default user should be changed immediately")
		log.Warn("This is the only time these credentials will be displayed")
	} else {
		log.Info("Created default user: " + userName + " for database: " + database)
	}
	return nil
}

func createJWTTable(database string) error {
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	wrapper, err := NewSQLiteWrapper(c.Storage.Path + "/" + database + ".db")
	if err != nil {
		log.Error("Error when creating JWT table: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	query := `CREATE TABLE IF NOT EXISTS ` + globals.JWTTable + ` (id TEXT PRIMARY KEY, token TEXT, sha256 TEXt, expiration TEXT)`
	// Create JWT table
	_, err = wrapper.Execute(query, globals.SystemUserID)
	if err != nil {
		log.Error("Error when creating JWT table: " + err.Error())
		if processor.FileDelete(dbFilePath) {
			log.Warn("Deleted database (" + database + ") due to failed initialization")
		}
		return errors.New(globals.ErrorDatabaseInitialization)
	}
	log.Debug("Successfully created system JWT table")
	return nil
}
