package db

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	log "github.com/sirupsen/logrus"
)

func Read(r *http.Request, w http.ResponseWriter, userID, correlationID string) {
	c.GetConfig()
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	sort := r.URL.Query().Get("sort")

	var erb globals.EntryRequest
	// Read the request body into a buffer
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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
	// Decode the buffer into requestBody
	err = json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&erb)
	if err != nil {
		log.Error("Failed to unmarshal request body: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		w.WriteHeader(500)
		response := globals.Response{
			Status:  "error",
			Message: "INTERNAL_SERVER_ERROR",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			return
		}
	}

	database := erb.Database

	// Reset the request body so it can be read again later
	r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	var data []map[string]interface{}

	for _, entry := range erb.Entries {
		table := entry.Table

		var field string
		var filters string
		for dataKey, dataValue := range entry.Data {
			if dataKey == "__select" {
				for _, selectFilter := range dataValue.([]interface{}) {

					filters = filters + selectFilter.(string) + ","
				}
			} else {
				field = dataKey
			}
		}
		filters = strings.TrimSuffix(filters, ",")

		log.Info("Using database: " + database + "/" + table + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

		var filterList []string
		var params []interface{}

		// Prepare the query with placeholders
		field = strings.TrimPrefix(field, "\"")
		field = strings.TrimSuffix(field, "\"")
		field = strings.TrimPrefix(field, "'")
		field = strings.TrimSuffix(field, "'")
		query := "SELECT " + field + " FROM " + table
		var args []interface{}

		if filters != "" {
			if strings.Contains(filters, "=") && !strings.Contains(filters, "'") {
				for _, filter := range strings.Split(filters, ",") {
					filter = strings.ReplaceAll(filter, "\"", "")
					var key, value string
					if strings.Contains(filter, "*") {
						filter = strings.ReplaceAll(filter, "*", "%")
						parts := strings.SplitN(filter, "=", 2)
						key = parts[0]
						value = parts[1]
						filterList = append(filterList, key+" LIKE ?")
					} else {
						parts := strings.SplitN(filter, "=", 2)
						key = parts[0]
						value = parts[1]
						filterList = append(filterList, key+" = ?")
					}
					params = append(params, value)
				}
			}
			query = strings.Join(filterList, " AND ")
			args = append(args, params...)
		}
		if sort != "" {
			query += " ORDER BY " + sort
		}
		if limit == "" && page != "" {
			log.Error("Invalid query parameters - Both limit and page must be provided together when using page" + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "Invalid query parameters - Both limit and page must be provided together when using page",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		if limit != "" {
			query += " LIMIT ?"
			args = append(args, limit)
		}
		if page != "" {
			query += " OFFSET ?"
			args = append(args, page)
		}

		log.Debug("Query: " + query)
		dbFilePath := c.Storage.Path + "/" + database + ".db"
		wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
		if err != nil {
			log.Fatal("Error when creating database wrapper: " + err.Error())
		}
		defer wrapper.Close()
		rows, err := wrapper.Query(query, args...)
		if err != nil {
			var responseMessage string
			var responseDataMap map[string]string
			switch err.Error() {
			case "no such table: " + table:
				log.Error("Table does not exist: " + table + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				responseMessage = "The table (" + table + ") does not exist in the database (" + database + ")"
				responseDataMap = map[string]string{"correlationID": correlationID}
			case "no such column: " + field:
				log.Error("Field does not exist: " + field + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				responseMessage = "The field (" + field + ") does not exist in the table (" + table + ")"
			default:
				log.Error("Failed to query database: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				responseMessage = "INTERNAL_SERVER_ERROR"
				responseDataMap = map[string]string{"correlationID": correlationID}
			}
			response := globals.Response{
				Status:  "error",
				Message: responseMessage,
				Data:    responseDataMap,
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

		var rowCount int
		for rows.Next() {
			columnPointers := make([]interface{}, len(columns))
			columnValues := make([]interface{}, len(columns))
			for i := range columnValues {
				columnPointers[i] = &columnValues[i]
			}
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
			rowCount++
		}

		data = append(data, map[string]interface{}{"rowCount": rowCount})
	}

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
	log.Info("Successfully queried database (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
}
