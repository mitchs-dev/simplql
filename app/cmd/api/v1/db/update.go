package db

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	"github.com/mitchs-dev/library-go/networking"
	log "github.com/sirupsen/logrus"
)

func Update(r *http.Request, w http.ResponseWriter, correlationID string) {
	c.GetConfig()
	var entryUpdate globals.EntryRequest

	// Decode the request body into the EntryUpdate struct
	err := json.NewDecoder(r.Body).Decode(&entryUpdate)
	if err != nil {
		log.Error("Failed to decode request body: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request body - Ensure that the request body is in the valid update format",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	log.Info("Using database: " + entryUpdate.Database + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	dbFilePath := c.Storage.Path + "/" + entryUpdate.Database + ".db"
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		log.Fatal("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	var sysEIDs []string

	for _, entry := range entryUpdate.Entries {
		table := entry.Table
		if strings.Contains(table, globals.SystemTablePrefix) {
			if strings.Contains(table, globals.UsersTable) {
				log.Error("Updating system tables (" + table + ") are prohibited - Please use the " + globals.NetworkingRequestAuthPath + " endpoint to update user information (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			}
			w.WriteHeader(400)
			log.Error("Updating system tables are prohibited (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "Updating system tables (" + table + ") are prohibited - Please use the " + globals.NetworkingRequestAuthPath + " endpoint to update user information",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		data := entry.Data

		// Log the data to debug
		log.Debug("Data: ", data)

		// Construct the update query
		var setClauses []string
		var setArgs []interface{}
		var filterClauses []string
		var filterArgs []interface{}

		// Check if the update key exists in the data map
		updateKey := globals.RequestUpdateParameter
		if updateValue, exists := data[updateKey]; exists {
			// Log the type of the update value
			log.Debug("Update Value Type: ", fmt.Sprintf("%T", updateValue))

			if updateFields, ok := updateValue.(map[string]interface{}); ok {
				// Log the updateFields to debug
				log.Debug("Update Fields: ", updateFields)

				for field, value := range updateFields {
					setClauses = append(setClauses, field+" = ?")
					setArgs = append(setArgs, value)
				}
			} else {
				// Log if updateFields is not of expected type
				log.Debug("Update Fields not of expected type")
			}
		} else {
			// Log if the update key is not found
			log.Debug("Update Key not found in data")
		}

		// Ensure setClauses is not empty
		if len(setClauses) == 0 {
			w.WriteHeader(400)
			log.Error("No fields to update (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "No fields to update",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}

		query := "UPDATE " + table + " SET " + strings.Join(setClauses, ", ")

		// Add filters if present
		for field, value := range data {
			if field == globals.RequestUpdateParameter {
				log.Debug("Skipping update key")
				continue
			} else {
				filterClauses = append(filterClauses, field+" = ?")
				log.Debug("Filter Field: ", field+" | Filter Value: ", value)
				filterArgs = append(filterArgs, value)
			}
		}
		if len(filterClauses) > 0 {
			query += " WHERE " + strings.Join(filterClauses, " AND ")
		}

		if len(setArgs)+len(filterArgs) != len(setClauses)+len(filterClauses) {
			w.WriteHeader(400)
			log.Error("Mismatch between filter values and filter clauses - Have: " + fmt.Sprint(len(setArgs)+len(filterArgs)) + " | Want: " + fmt.Sprint(len(setClauses)+len(filterClauses)) + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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

		log.Debug("Update Query: " + query + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

		// Execute the update query
		_, err = wrapper.Execute(query, append(setArgs, filterArgs...)...)
		if err != nil {
			log.Error("Failed to execute update query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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

		// Retrieve the affected sys_eid values
		selectQuery := "SELECT sys_eid FROM " + table
		if len(filterClauses) > 0 {
			selectQuery += " WHERE " + strings.Join(filterClauses, " AND ")
		}
		log.Debug("Select Query: " + selectQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

		rows, err := wrapper.Query(selectQuery, filterArgs...)
		if err != nil {
			log.Error("Failed to execute select query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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
		defer rows.Close()

		for rows.Next() {
			var sysEID string
			if err := rows.Scan(&sysEID); err != nil {
				log.Error("Failed to scan row: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				continue
			}
			sysEIDs = append(sysEIDs, sysEID)
		}
	}

	response := globals.Response{
		Status:  "success",
		Message: "ENTRY_UPDATED",
		Data: map[string]interface{}{
			"correlationID": correlationID,
			"sys_eids":      sysEIDs,
		},
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
	return
}
