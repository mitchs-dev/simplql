package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/data"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	log "github.com/sirupsen/logrus"
)

func Read(r *http.Request, w http.ResponseWriter, userID, correlationID string) {

	// Read the request body
	requestBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		response := globals.Response{
			Status:  "error",
			Message: "Failed to read request body",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	arb.GetAuthRequest(requestBody, correlationID)

	c.GetConfig()
	database := arb.Database
	table := globals.UsersTable
	page := r.URL.Query().Get("page")
	limit := r.URL.Query().Get("limit")
	sort := r.URL.Query().Get("sort")

	var field string
	var args []interface{}
	if len(arb.Data.Select) == 0 {
		log.Error("Invalid query parameters - Select must be provided" + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid query parameters - Select field(s) (" + globals.RequestSelectParameter + ") must be provided in request",
			Data:    map[string]string{"correlationID": correlationID},
		}
		w.WriteHeader(http.StatusBadRequest)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	} else {
		field = strings.Join(arb.Data.Select, ",")
	}

	var filtersAsList []string
	if arb.Data.ID != "" {
		filtersAsList = append(filtersAsList, "id = '"+arb.Data.ID+"'")
	}
	if arb.Data.Name != "" {
		filtersAsList = append(filtersAsList, "name = '"+arb.Data.Name+"'")
	}
	if arb.Data.Password != "" {
		filtersAsList = append(filtersAsList, "password = '"+arb.Data.Password+"'")
	}
	if len(arb.Data.Roles) > 0 {
		var rolesList []string
		for _, role := range arb.Data.Roles {
			if strings.HasPrefix(role, "*") && strings.HasSuffix(role, "*") || strings.HasPrefix(role, "%") || strings.HasSuffix(role, "%") {
				role = strings.TrimPrefix(role, "*")
				role = strings.TrimSuffix(role, "*")
				if !strings.HasPrefix(role, "%") && !strings.HasSuffix(role, "%") {
					role = "%" + role + "%"
				}
				rolesList = append(rolesList, "roles LIKE '"+role+"'")
			} else {
				rolesList = append(rolesList, "roles= ? ")
				args = append(args, role)
			}
		}
		filtersAsList = append(filtersAsList, strings.Join(rolesList, " AND "))
	}
	filters := strings.Join(filtersAsList, " AND ")

	log.Info("Using database: " + database + "/" + table + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	// Prepare the query with placeholders
	field = strings.TrimPrefix(field, "\"")
	field = strings.TrimSuffix(field, "\"")
	field = strings.TrimPrefix(field, "'")
	field = strings.TrimSuffix(field, "'")
	query := "SELECT " + field + " FROM " + table

	if filters != "" {
		if strings.Contains(filters, "=") && !strings.Contains(filters, "'") {
			var filterList []string
			for _, filter := range strings.Split(filters, ",") {
				filter = strings.ReplaceAll(filter, "\"", "")
				if strings.Contains(filter, "*") {
					filter = strings.ReplaceAll(filter, "*", "%")
					filter = strings.ReplaceAll(filter, "=", " LIKE '")
					args = append(args, filter)
				} else {
					filter = strings.ReplaceAll(filter, "=", " = '")
				}
				filter = filter + "'"
				filterList = append(filterList, filter)
			}
			filters = strings.Join(filterList, " AND ")
		}
		query += " WHERE " + strings.TrimSuffix(filters, " AND ")
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

	var rowCount int

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

	var dataMap []map[string]interface{}
	for rows.Next() {
		log.Debug("Processing row: ", rowCount)
		columnPointers := make([]interface{}, len(columns))
		columnValues := make([]interface{}, len(columns))
		for i := range columnValues {
			columnPointers[i] = &columnValues[i]
		}

		if err := rows.Scan(columnPointers...); err != nil {
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

		for i := range columnValues {
			log.Debugf("Type of columnValues[%d]: %T\n", i, columnValues[i]) // Debugging line

			var val string
			switch v := columnValues[i].(type) {
			case string:
				val = v
			case []byte:
				val = string(v)
			default:
				val = fmt.Sprintf("%v", v)
			}

			if val != "" {
				processedVal := data.Process(val)
				columnValues[i] = processedVal // Store the processed value back into columnValues
			} else {
				log.Error("Failed to convert column value to string", " (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
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
		}

		rowData := make(map[string]interface{})
		for i, colName := range columns {
			rowData[colName] = columnValues[i]
		}
		dataMap = append(dataMap, rowData)
		rowCount++
	}

	dataMap = append(dataMap, map[string]interface{}{"rowCount": rowCount})

	// Send the data back as a JSON response
	response := globals.Response{
		Status:  "success",
		Message: "QUERY_SUCCESS",
		Data:    dataMap,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
	log.Info("Successfully queried database (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
}
