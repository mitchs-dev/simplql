package sqlWrapper

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"

	log "github.com/sirupsen/logrus"

	"github.com/mitchs-dev/library-go/generator"
	"github.com/mitchs-dev/library-go/processor"

	_ "github.com/mattn/go-sqlite3"
)

var c configuration.Configuration

var skipDataProcess bool

// SQLiteWrapper is a struct that holds the database connection
type SQLiteWrapper struct {
	db   *sql.DB
	name string
}

// NewSQLiteWrapper creates a new SQLiteWrapper and enables WAL mode
func NewSQLiteWrapper(dataSourceName string) (*SQLiteWrapper, error) {
	db, err := sql.Open("sqlite3", dataSourceName)
	if err != nil {
		return nil, err
	}

	// Ensure that each wrapper has WAL mode enabled
	_, err = db.Exec("PRAGMA journal_mode = WAL;")
	if err != nil {
		return nil, err
	}

	databaseName := filepath.Base(dataSourceName)
	databaseNameExt := filepath.Ext(databaseName)
	databaseName = strings.TrimSuffix(databaseName, databaseNameExt)
	return &SQLiteWrapper{db: db, name: databaseName}, nil
}

// Close closes the database connection
func (wrapper *SQLiteWrapper) Close() error {
	return wrapper.db.Close()
}

// Execute executes a query without returning any rows
func (wrapper *SQLiteWrapper) Execute(query, userID string, args ...interface{}) (sql.Result, error) {
	tx, err := wrapper.db.BeginTx(context.Background(), &sql.TxOptions{
		Isolation: sql.LevelSerializable,
	})
	if err != nil {
		return nil, err
	}

	stmt, err := tx.Prepare(query)
	if err != nil {
		tx.Rollback()
		return nil, err
	}
	defer stmt.Close()

	var newArgs []interface{}
	if strings.Contains(query, "DELETE") {
		for _, arg := range args {
			switch v := arg.(type) {
			case []string:
				for _, str := range v {
					newArgs = append(newArgs, str)
				}
			default:
				newArgs = append(newArgs, arg)
			}
		}
	} else {
		for _, arg := range args {
			switch v := arg.(type) {
			case []string:
				for _, str := range v {
					newArg := data.Process(str)
					newArgs = append(newArgs, newArg)
				}
			default:
				newArg := data.Process(arg)
				newArgs = append(newArgs, newArg)
			}
		}
	}

	var oldValues string
	if strings.Contains(query, "UPDATE") {
		oldValueColumns := fetchSetColumns(query)
		if oldValueColumns != "" {
			// Extract the columns and arguments from the query
			var oldValueWhereStatement string

			// Split the query into SET and WHERE parts
			setClauseIndex := strings.Index(query, "SET")
			whereClauseIndex := strings.Index(query, "WHERE")

			// Ensure that both SET and WHERE clauses exist
			var (
				setClausePlaceholderCount   int
				whereClausePlaceholderCount int
				setClause                   string
				whereClause                 string
			)
			if setClauseIndex != -1 && whereClauseIndex != -1 && whereClauseIndex > setClauseIndex {
				setClause = query[setClauseIndex+len("SET") : whereClauseIndex]
				whereClause = query[whereClauseIndex+len("WHERE"):]

				// Count the number of placeholders in each clause
				setClausePlaceholderCount = strings.Count(setClause, "?")
				whereClausePlaceholderCount = strings.Count(whereClause, "?")
			} else {
				return nil, errors.New("error when creating transaction: SET and WHERE clauses could not be determined or are invalid")
			}
			log.Debug("Set clause placeholder count: " + fmt.Sprint(setClausePlaceholderCount))
			log.Debug("Where clause placeholder count: " + fmt.Sprint(whereClausePlaceholderCount))

			// Separate the arguments based on the counts
			setClauseArgs := newArgs[:setClausePlaceholderCount]
			whereClauseArgs := newArgs[setClausePlaceholderCount:]
			log.Debug("Set clause args: " + fmt.Sprintf("%v", setClauseArgs))
			log.Debug("Where clause args: " + fmt.Sprintf("%v", whereClauseArgs))

			// Construct the WHERE statement
			rawFilters := strings.Split(whereClause, "AND")
			var filters []string
			for _, rawFilter := range rawFilters {
				filters = append(filters, strings.TrimSpace(rawFilter))
			}
			oldValueWhereStatement = " WHERE " + strings.Join(filters, " AND ")

			// Use the columns to fetch the old values
			oldValuesQuery := "SELECT " + oldValueColumns + " FROM " + extractTableName(query) + oldValueWhereStatement
			log.Debug("Old values query: " + oldValuesQuery + " with args: " + fmt.Sprintf("%v", whereClauseArgs) + " (arg count: " + fmt.Sprint(len(whereClauseArgs)) + ")")
			skipDataProcess = true
			oldValuesRow := wrapper.QueryRow(oldValuesQuery, whereClauseArgs...)
			skipDataProcess = false

			// Split the columns by comma and create a slice of interface{} to hold the values
			columns := strings.Split(oldValueColumns, ",")
			values := make([]interface{}, len(columns))
			valuePointers := make([]interface{}, len(columns))
			for i := range values {
				valuePointers[i] = &values[i]
			}

			// Add debug logs to inspect the state before calling Scan
			log.Debugf("Columns: %v", columns)
			log.Debugf("Values: %v", values)
			log.Debugf("Value Pointers: %v", valuePointers)
			log.Debugf("Executing query: %s with args: %v", oldValuesQuery, whereClauseArgs)

			err := oldValuesRow.Scan(valuePointers...)
			if err != nil {
				if strings.Contains(err.Error(), "no rows in result set") {
					log.Debug("Error creating transaction - It's possible that the old values are empty: " + err.Error())
					log.Debugf("Query: %s with args: %v", oldValuesQuery, whereClauseArgs)
					return nil, errors.New(globals.ErrorTransactionNoEntry)
				} else {
					log.Error("Error when fetching old values: " + err.Error())
					return nil, err
				}
			}

			// Add debug logs to inspect the values after calling Scan
			for i, col := range columns {
				log.Debugf("Column: %s, Value: %v", col, values[i])
			}

			// Convert the values to a string representation
			oldValues = fmt.Sprintf("%v", values)
		}
	}

	log.Debug("Executing query: " + query + " with args: " + fmt.Sprintf("%v", newArgs))
	result, err := stmt.Exec(newArgs...)
	var (
		action    string
		table     string
		recordID  string
		newValues string
		ipAddress string
		status    string
	)
	table = extractTableName(query)
	if table == "" {
		return nil, errors.New(globals.ErrorTransactionTableNameExtraction)
	}
	if table == globals.TransactionsTable {
		log.Debug("Transaction table detected - skipping transaction creation")
		globals.IsTransactionExecution = true
	} else {
		log.Debug("Transaction table not detected - creating transaction")
		globals.IsTransactionExecution = false
		recordID = extractRecordID(query, newArgs)

		if recordID == "" && !strings.Contains(query, "CREATE") && !strings.Contains(query, globals.SystemTablePrefix) {
			return nil, errors.New(globals.ErrorTransactionRecordIDExtraction)
		}
		ipAddress = ""
		status = "ERROR"
	}
	switch {
	case strings.Contains(query, "INSERT"):
		action = "INSERT"
		newValues = fmt.Sprintf("%v", newArgs)
	case strings.Contains(query, "UPDATE"):
		action = "UPDATE"
		newValues = fmt.Sprintf("%v", newArgs)
	case strings.Contains(query, "DELETE"):
		action = "DELETE"
		newValues = fmt.Sprintf("%v", newArgs)
	case strings.Contains(query, "SELECT"):
		action = "SELECT"
	case strings.Contains(query, "CREATE"):
		action = "CREATE"
	default:
		log.Error("Unknown action in query: " + strings.Split(query, " ")[0])
		action = "UNKNOWN"
	}

	if err != nil {
		tx.Rollback()
		if action != "SELECT" && action != "UNKNOWN" {
			action = action + "(ROLLBACK)"
			if !globals.IsTransactionExecution {
				err = createTransaction(wrapper.name, userID, action, table, recordID, oldValues, newValues, ipAddress, status, err)
			} else {
				log.Debug("Execution is for a transaction - not creating a transaction")
			}
		}
		return nil, err
	}

	err = tx.Commit()
	if err != nil {
		if action != "SELECT" && action != "UNKNOWN" {
			status = "ERROR"
			if !globals.IsTransactionExecution {
				err = createTransaction(wrapper.name, userID, action, table, recordID, oldValues, newValues, ipAddress, status, err)
			} else {
				log.Debug("Execution is for a transaction - not creating a transaction")
			}
		}
		return nil, err
	}

	if action != "SELECT" && action != "UNKNOWN" {
		status = "SUCCESS"
		if !globals.IsTransactionExecution {
			err = createTransaction(wrapper.name, userID, action, table, recordID, oldValues, newValues, ipAddress, status, err)
		} else {
			log.Debug("Execution is for a transaction - not creating a transaction")
		}
		if err != nil {
			return nil, err
		}
	}

	return result, nil
}

// Query executes a query that returns rows
func (wrapper *SQLiteWrapper) Query(query string, args ...interface{}) (*sql.Rows, error) {
	var filterArgs []interface{}
	var newArgs []interface{}

	// Regular expressions to find patterns like 'LIKE <ARG>' and '= <ARG>'
	likeRe := regexp.MustCompile(`LIKE\s+'[^']*'`)
	equalRe := regexp.MustCompile(`=\s+'[^']*'`)

	// Find all matches for LIKE patterns
	likeMatches := likeRe.FindAllString(query, -1)
	for _, match := range likeMatches {
		query = strings.Replace(query, match, "LIKE ?", 1)
		arg := match[6 : len(match)-1] // Extract the argument without LIKE and quotes
		filterArgs = append(filterArgs, arg)
	}

	// Find all matches for = patterns
	equalMatches := equalRe.FindAllString(query, -1)
	for _, match := range equalMatches {
		query = strings.Replace(query, match, "= ?", 1)
		arg := match[3 : len(match)-1] // Extract the argument without = and quotes
		filterArgs = append(filterArgs, arg)
	}

	args = append(args, filterArgs...)

	// Process the remaining arguments
	for _, arg := range args {
		var newArg interface{}
		var isLike bool

		switch v := arg.(type) {
		case string:
			log.Debug("Processing string argument: " + v)
			if strings.Contains(v, "%") {
				v = strings.ReplaceAll(v, "%", "")
				isLike = true
				log.Debug("Argument contains %: " + v)
			} else {
				log.Debug("Argument does not contain %: " + v)
				isLike = false
			}
			newArg = data.Process(v)
			if isLike {
				newArg = "%" + newArg.(string) + "%"
			}
		case []string:
			log.Debug("Processing []string argument")
			for i, s := range v {
				if strings.Contains(s, "%") {
					v[i] = strings.ReplaceAll(s, "%", "")
					isLike = true
					log.Debug("Argument contains %: " + s)
				} else {
					log.Debug("Argument does not contain %: " + s)
					isLike = false
				}
				processed := data.Process(v[i])
				if isLike {
					v[i] = "%" + processed.(string) + "%"
				} else {
					v[i] = processed.(string)
				}
			}
			newArg = v
		default:
			log.Debug("Processing unknown type argument")
			newArg = data.Process(arg)
		}

		newArgs = append(newArgs, newArg)
	}
	log.Debug("Running query: " + query)
	log.Debug("New args: ", newArgs)

	rows, err := wrapper.db.Query(query, newArgs...)

	userID := "toBeUpdated"
	action := "SELECT"
	if c.Logging.Transactions.LogSelectQueries {
		log.Warn("SELECT query logging is not supported yet")
		return rows, err
		table := extractTableName(query)
		if table == "" {
			return nil, errors.New(globals.ErrorTransactionTableNameExtraction)
		}
		recordID := extractRecordID(query, newArgs)
		if recordID == "" {
			return nil, errors.New(globals.ErrorTransactionRecordIDExtraction)
		}
		oldValues := ""
		newValues := ""
		ipAddress := ""
		var status string
		if err != nil {
			status = "ERROR"
		} else {
			status = "SUCCESS"
		}

		if !globals.IsTransactionExecution {
			err = createTransaction(wrapper.name, userID, action, table, recordID, oldValues, newValues, ipAddress, status, err)
		} else {
			log.Debug("Execution is for a transaction - not creating a transaction")
		}
		if err != nil {
			return nil, err
		}
	}

	return rows, err
}

// QueryRow executes a query that returns a single row
func (wrapper *SQLiteWrapper) QueryRow(query string, args ...interface{}) *sql.Row {
	var newArgs []interface{}
	if !skipDataProcess {
		log.Debug("Processing arguments")
		for _, arg := range args {
			newArg := data.Process(arg)
			newArgs = append(newArgs, newArg)
		}
	} else {
		log.Debug("Skipping data processing")
		newArgs = args
	}
	return wrapper.db.QueryRow(query, newArgs...)
}

// createDatabases creates the databases specified in the configuration
func CreateDatabases() error {
	c.GetConfig()
	log.Debug("Initializing databases")
	for _, database := range c.Databases {
		log.Debug("Initializing database: " + database.Name)
		dbFilePath := c.Storage.Path + "/" + database.Name + ".db"
		wrapper, err := NewSQLiteWrapper(dbFilePath)
		if err != nil {
			log.Fatal("Error when creating database wrapper: " + err.Error())
		}
		defer wrapper.Close()
		var dbVersionRaw string
		var createDB bool
		err = wrapper.QueryRow("SELECT version FROM " + globals.MetadataTable).Scan(&dbVersionRaw)
		if err != nil {
			if strings.Contains(err.Error(), "no such table: "+globals.MetadataTable) {
				log.Debug("Metadata table does not exist: " + globals.MetadataTable + " - Will create it")
				createDB = true
			} else if strings.Contains(err.Error(), "no such file or directory") {
				log.Debug("Database file does not exist: " + dbFilePath + " - Will create it")
				createDB = true
			} else {
				log.Fatal("Error when querying database version: " + err.Error())
			}
		} else {
			createDB = false
		}
		if !processor.DirectoryOrFileExists(dbFilePath) {
			log.Info("Creating database file: " + dbFilePath)
			if !processor.CreateFileAsByte(dbFilePath, []byte{}) {
				log.Fatal("Error when creating database file: " + dbFilePath)
			}
		} else {
			log.Debug("Database file already exists: " + dbFilePath)
		}
		if !createDB {
			dbVersion, err := strconv.Atoi(data.Process(dbVersionRaw).(string))
			if err != nil {
				log.Fatal("Error when converting database version: " + err.Error())
			}
			if dbVersion != database.Version {
				log.Warn("Database version mismatch: " + dbFilePath)
			} else {
				log.Info("Database " + database.Name + "@v" + fmt.Sprint(dbVersion) + " is ready")

			}
		} else {
			log.Debug("Creating database: " + database.Name)
			err = createTransactionsTable(database.Name)
			if err != nil {
				return err
			}
			err = createMetadataTable(database.Name, database.Version)
			if err != nil {
				return err
			}
			err = createUsersTable(database.Name)
			if err != nil {
				return err
			}
			err = createJWTTable(database.Name)
			if err != nil {
				return err
			}
			// Create tables
			log.Debug("Creating database tables for: " + database.Name)
			for _, table := range database.Tables {
				if strings.HasPrefix(table.Name, globals.SystemTablePrefix) {
					log.Error("Invalid table name: " + table.Name + " - Table names cannot be prefixed with '" + globals.SystemTablePrefix + "' as this is reserved for system tables")
					if processor.FileDelete(dbFilePath) {
						log.Warn("Deleted database (" + database.Name + ") due to failed initialization")
					}
					return errors.New(globals.ErrorInvalidTableName)
				}
				columns := make([]string, 0)
				for _, column := range table.Columns {
					var columnTypeIsValid bool
					for _, columnType := range globals.ColumnTypes {
						if column.Type == columnType {
							if strings.HasPrefix(column.Name, globals.SystemColumnPrefix) {
								log.Error("Invalid column name: " + column.Name + " - Column names cannot be prefixed with '" + globals.SystemColumnPrefix + "' as this is reserved for system columns")
								if processor.FileDelete(dbFilePath) {
									log.Warn("Deleted database (" + database.Name + ") due to failed initialization")
								}
								return errors.New(globals.ErrorDatabaseInitialization)
							}
							columnTypeIsValid = true
							break
						} else {
							columnTypeIsValid = false
						}
					}
					if !columnTypeIsValid {
						log.Error("Invalid column type: " + column.Type + " for column: " + column.Name + " in table: " + table.Name)
						log.Info("Columns must be a text-type. Valid types are: " + strings.Join(globals.ColumnTypes, ", "))
						if processor.FileDelete(dbFilePath) {
							log.Warn("Deleted database (" + database.Name + ") due to failed initialization")
						}
						return errors.New(globals.ErrorInvalidColumnType)
					}
					columns = append(columns, column.Name+" "+column.Type)
					if column.PrimaryKey {
						columns[len(columns)-1] += " PRIMARY KEY"
					}
				}
				// Insert entry ID column
				columns = append(columns, globals.TableEntryIDColumnName+" TEXT")
				query := `CREATE TABLE IF NOT EXISTS ` + table.Name + ` (` + strings.Join(columns, ", ") + `)`
				_, err := wrapper.Execute(query, globals.SystemUserID)
				if err != nil {
					log.Error("Error when creating table: " + err.Error())
					if processor.FileDelete(dbFilePath) {
						log.Warn("Deleted database (" + database.Name + ") due to failed initialization")
					}
					return errors.New(globals.ErrorDatabaseInitialization)
				}
			}
		}
	}
	log.Info("Databases initialized")
	return nil
}

// Creates a transaction in the database
func createTransaction(database, userID, actionType, affectedTable, recordID, oldValues, newValues, ipAddress, status string, errorMessage error) error {

	c.GetConfig()

	if !c.Logging.Transactions.Enabled {
		globals.IsTransactionExecution = false
		log.Debug("Transactions are disabled - skipping transaction creation")
		return nil
	}

	wrapper, err := NewSQLiteWrapper(c.Storage.Path + "/" + database + ".db")
	if err != nil {
		return errors.New("Error when creating transaction: " + err.Error())
	}
	defer wrapper.Close()
	// If any of the values are empty, set them to NULL
	if userID == "" {
		userID = "NULL"
	}
	if actionType == "" {
		return errors.New("error when creating transaction: actionType cannot be empty")
	} else if actionType == "SELECT" {
		if !c.Logging.Transactions.LogSelectQueries {
			globals.IsTransactionExecution = false
			log.Debug("Transactions are disabled for SELECT queries - skipping transaction creation")
			return nil
		}
	}
	log.Debug("Action type: " + actionType)

	if affectedTable == "" {
		return errors.New("error when creating transaction: affectedTable cannot be empty")
	} else if affectedTable == globals.ErrorTransaction {
		return errors.New("error when creating transaction: affectedTable could not be determined")
	}
	if recordID == "" {
		if actionType == "CREATE" {
			if strings.Contains(affectedTable, globals.TransactionsTable) {
				log.Debug("Transaction is for a transaction table - skipping this to avoid chicken-and-egg scenario")
				return nil
			}
			log.Debug("Transaction is for a create action - generating recordID")
			recordID = "create-" + affectedTable + "-" + generator.RandomString(globals.TableEntryIDLength)
		} else if strings.Contains(affectedTable, globals.SystemTablePrefix) {
			log.Debug("Transaction is for a system table - generating recordID")
			recordID = "system-" + actionType + "-" + generator.RandomString(globals.TableEntryIDLength)
		} else {
			return errors.New("error when creating transaction: recordID cannot be empty")
		}
	}
	if oldValues == "" {
		oldValues = "NULL"
	}
	if newValues == "" {
		newValues = "NULL"
	}
	if ipAddress == "" {
		ipAddress = "NULL"
	}
	if status == "" {
		return errors.New("error when creating transaction: status cannot be empty")
	}
	var errMessageString string
	if errorMessage != nil {
		errMessageString = errorMessage.Error()
	} else {
		errMessageString = "NULL"
	}
	timestamp := generator.Timestamp("Local")
	query := `INSERT INTO ` + globals.TransactionsTable + ` (Timestamp, userID, actionType, affectedTable, recordID, oldValues, newValues, ipAddress, status, errorMessage) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`
	args := []interface{}{timestamp, userID, actionType, affectedTable, recordID, oldValues, newValues, ipAddress, status, errMessageString}
	globals.IsTransactionExecution = true
	log.Debug("Transaction creation query: " + query)
	log.Debug("Transaction creation args: ", args)
	_, err = wrapper.Execute(query, globals.SystemUserID, args...)
	globals.IsTransactionExecution = false
	if err != nil {
		log.Error("Error when creating transaction: " + err.Error())
		return err
	}
	log.Debug("Transaction created for " + recordID)
	return nil
}
