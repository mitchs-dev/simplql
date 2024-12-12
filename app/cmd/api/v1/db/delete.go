package db

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	log "github.com/sirupsen/logrus"
)

func Delete(r *http.Request, w http.ResponseWriter, userID, correlationID string) {
	c.GetConfig()

	database := r.URL.Query().Get("database")
	table := r.URL.Query().Get("table")
	filters := r.URL.Query().Get("filters")

	log.Info("Using database: " + database + "/" + table + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	selectQuery := "SELECT " + globals.TableEntryIDColumnName + " FROM " + table
	var args []interface{}

	if filters != "" {
		if strings.Contains(filters, "=") && !strings.Contains(filters, "'") {
			var filterList []string
			for _, filter := range strings.Split(filters, ",") {
				filter = strings.ReplaceAll(filter, "\"", "")
				if strings.Contains(filter, "*") {
					filter = strings.ReplaceAll(filter, "*", "%")
					filter = strings.ReplaceAll(filter, "=", " LIKE '")
					filter = strings.ReplaceAll(filter, "' ", "'")
					filter = strings.ReplaceAll(filter, " '", "'")
				} else {
					filter = strings.ReplaceAll(filter, "=", " = '")
				}
				filter = filter + "'"
				filterList = append(filterList, filter)
			}
			filters = strings.Join(filterList, " AND ")
		}
		selectQuery += " WHERE " + strings.TrimSuffix(filters, " AND ")
	}

	log.Debug("Select Query: " + selectQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	dbFilePath := c.Storage.Path + "/" + database + ".db"
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		log.Fatal("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	// Execute the select query
	rows, err := wrapper.Query(selectQuery, args...)
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

	// Get the data
	columns, err := rows.Columns()
	if err != nil {
		log.Error("Failed to get columns", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
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

	var (
		data     []map[string]interface{}
		rowCount int
	)

	columnValues := make([]interface{}, len(columns))
	columnPointers := make([]interface{}, len(columns))

	for rows.Next() {
		err = rows.Scan(columnPointers...)
		if err != nil {
			log.Error("Failed to scan row", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
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
		rowData := make(map[string]interface{})
		for i, colName := range columns {
			rowData[colName] = columnValues[i]
		}
		data = append(data, rowData)
		log.Info("Selected row: " + rowData[globals.TableEntryIDColumnName].(string) + " from database: " + database + "/" + table + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		rowCount++
	}

	var (
		delArgs         []interface{}
		argPlaceholders string
	)
	if len(data) == 0 {
		log.Error("No rows found in table: " + table + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(400)
		response := globals.Response{
			Status:  "error",
			Message: "No rows found in table: " + table,
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}
	for _, row := range data {
		log.Debug("Processing row: ", row)
		if syseid, ok := row[globals.TableEntryIDColumnName].(string); ok {
			delArgs = append(delArgs, syseid)
			argPlaceholders = argPlaceholders + ", ?"
		} else {
			// Debugging: Print a warning if the ID is not found or not a string
			log.Debug("ID not found or not a string in row: ", row)
		}
	}

	deleteQuery := "DELETE FROM " + table

	deleteQuery += " WHERE " + globals.TableEntryIDColumnName + " IN (" + strings.TrimPrefix(argPlaceholders, ",") + ")"

	log.Debug("Delete Query: " + deleteQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	// Execute the delete query
	_, err = wrapper.Execute(deleteQuery, userID, delArgs...)
	if err != nil {
		log.Error("Failed to execute delete query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(400)
		response := globals.Response{
			Status:  "error",
			Message: "Failed to execute delete query - Ensure that the query is valid",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	data = append(data, map[string]interface{}{"rowCount": rowCount})

	// Send the data back as a JSON response
	response := globals.Response{
		Status:  "success",
		Message: "QUERY_SUCCESS",
		Data:    data,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
}
