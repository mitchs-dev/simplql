package db

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mitchs-dev/library-go/generator"
	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	log "github.com/sirupsen/logrus"
)

var c configuration.Configuration

func Create(r *http.Request, w http.ResponseWriter, userID, correlationID string) {
	c.GetConfig()
	// Read request body
	var requestBody globals.EntryRequest
	err := json.NewDecoder(r.Body).Decode(&requestBody)
	if err != nil {
		log.Error("Failed to unmarshal request body: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request body - Ensure that the request body is in the valid creation format",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}
	database := requestBody.Database
	if database == "" {
		log.Error("Failed to read database name", "Database name is empty"+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid database name - Ensure that the database name is not empty",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}
	log.Debug("Using database: " + database + " for entry creation (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	var entryIDs []string
	var entryIndices []int
	var tableNames []string
	for entryIndex, entry := range requestBody.Entries {
		entryIndices = append(entryIndices, entryIndex)
		entryID := globals.TableEntryIDPrefix + generator.RandomString(globals.TableEntryIDLength) + globals.TableEntryIDSuffix
		entryIDs = append(entryIDs, entryID)
		tableName := entry.Table
		tableNames = append(tableNames, tableName)

		// Ensure that an exact entry does not already exist
		var columns []string
		var params []interface{}
		for key, value := range entry.Data {
			columns = append(columns, key)
			if strSlice, ok := value.([]string); ok {
				log.Debug("Value is a slice, converting to JSON")
				// Convert roles slice to JSON string
				valueJSON, err := json.Marshal(strSlice)
				if err != nil {
					log.Fatalf("Failed to marshal roles: %v", err)
				}

				log.Debug("Key in JSON: ", string(key), " for user creation (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				value = string(valueJSON)
			}
			params = append(params, value)
		}
		query := "SELECT " + globals.TableEntryIDColumnName + " FROM " + tableName + " WHERE " + strings.Join(columns, " = ? AND ") + " = ?"
		dbFilePath := c.Storage.Path + "/" + database + ".db"
		wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
		if err != nil {
			log.Fatal("Error when creating database wrapper: " + err.Error())
		}
		defer wrapper.Close()
		log.Debug("Query: " + query + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		log.Debug("Params: ", params, " (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		rows, err := wrapper.Query(query, params...)
		if err != nil {
			log.Error("Failed to execute select query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "Failed to execute select query - Ensure that the query is valid",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		defer rows.Close()

		// Check if the entry already exists
		log.Debug("Checking for exact entry in table: " + tableName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		if rows.Next() {
			err := rows.Scan(&entryID)
			if err != nil {
				log.Error("Failed to scan row: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				response := globals.Response{
					Status:  "error",
					Message: "INTERNAL_SERVER_ERROR",
					Data:    map[string]string{"correlationID": correlationID},
				}
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
			dataMatch := data.Process(entryID).(string)
			log.Error("An exact match (" + dataMatch + ") for the entry already exists in table: " + tableName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			w.WriteHeader(409)
			response := globals.Response{
				Status:  "error",
				Message: "An exact match (" + dataMatch + ") for the entry already exists in the table",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err = json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}

		log.Debug("Creating entry in table: " + tableName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		data := entry.Data
		log.Info("Creating entry (" + entryID + ") in table: " + tableName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		// Create entry
		query = "INSERT INTO " + tableName + " ( " + globals.TableEntryIDColumnName + " ) VALUES ( ? )"
		dbFilePath = c.Storage.Path + "/" + database + ".db"
		wrapper, err = sqlWrapper.NewSQLiteWrapper(dbFilePath)
		if err != nil {
			log.Fatal("Error when creating database wrapper: " + err.Error())
		}
		defer wrapper.Close()
		log.Debug("Query: " + query + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		_, err = wrapper.Execute(query, userID, entryID)
		if err != nil {
			log.Error("Failed to create entry: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "INTERNAL_SERVER_ERROR",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		var args []interface{}
		log.Debug("Created row for entry: " + entryID + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		for key, _ := range data {
			query := "UPDATE " + tableName + " SET " + key + " = ? WHERE " + globals.TableEntryIDColumnName + " = ?"
			if strings.Contains(key, ",") {
				log.Debug("Value is a slice, converting to JSON")
				// Convert roles slice to JSON string
				keyJSON, err := json.Marshal(key)
				if err != nil {
					log.Fatalf("Failed to marshal roles: %v", err)
				}

				log.Debug("Key in JSON: ", string(key), " for user creation (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				key = string(keyJSON)
			}
			args = append(args, data[key])
			args = append(args, entryID)

			dbFilePath := c.Storage.Path + "/" + database + ".db"
			wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
			if err != nil {
				log.Fatal("Error when creating database wrapper: " + err.Error())
			}
			defer wrapper.Close()
			_, err = wrapper.Execute(query, userID, args...)
			if err != nil {
				log.Error("Failed to create entry: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				w.WriteHeader(http.StatusInternalServerError)
				response := globals.Response{
					Status:  "error",
					Message: "INTERNAL_SERVER_ERROR",
					Data:    map[string]string{"correlationID": correlationID},
				}
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
			log.Debug("Created column " + key + " for entry: " + entryID + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		}
		log.Info("Created entry (" + entryID + ") in table: " + tableName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	}
	log.Info("All entries processed successfully (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	responseReceipt := globals.EntryCreationResponse{}
	for currentIndex := range entryIndices {
		responseReceipt.EntryReceipts = append(responseReceipt.EntryReceipts, globals.EntryCreationResponseEntryReceipt{
			EntryID:      entryIDs[currentIndex],
			Table:        tableNames[currentIndex],
			RequestIndex: entryIndices[currentIndex],
		})
	}
	response := globals.Response{
		Status:  "ok",
		Message: "Entry created",
		Data:    responseReceipt,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
}
