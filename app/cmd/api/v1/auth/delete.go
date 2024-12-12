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

func Delete(r *http.Request, w http.ResponseWriter, userID, correlationID string) {
	c.GetConfig()

	authBody, err := io.ReadAll(r.Body)
	if err != nil {
		log.Error("Failed to read request body: " + err.Error() + " (C: " + correlationID + ")")
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

	arb.GetAuthRequest(authBody, correlationID)

	database := arb.Database
	table := globals.UsersTable
	var filtersAsList []string
	id := arb.Data.ID
	password := arb.Data.Password
	name := arb.Data.Name
	roles := arb.Data.Roles
	var args []interface{}

	if id != "" {
		filtersAsList = append(filtersAsList, "id= ? ")
		args = append(args, id)
	}
	if name != "" {
		if strings.Contains(name, "*") {
			name = strings.ReplaceAll(name, "*", "%")
			name = "name LIKE '" + name + "'"
			filtersAsList = append(filtersAsList, name)
		} else {
			filtersAsList = append(filtersAsList, "name= ? ")
			args = append(args, name)
		}
	}
	if password != "" {
		filtersAsList = append(filtersAsList, "password= ? ")
		args = append(args, password)
	}
	if len(roles) > 0 {
		if len(roles) == 1 && roles[0] == "" {
			log.Warn("Empty roles provided - Ignoring roles filter (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		} else {
			rolesString := strings.Join(roles, ",")
			filtersAsList = append(filtersAsList, "roles= ? ")
			args = append(args, rolesString)
		}
	}

	filters := strings.Join(filtersAsList, " AND ")
	if filters == "" {
		log.Error("No filters provided for delete query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "No filters provided for delete query - Ensure that you are providing at least one filter",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	log.Info("Using database: " + database + "/" + table + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	selectQuery := "SELECT " + globals.UserEntryIDColumnName + ",roles FROM " + table

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

	if rows == nil {
		log.Warn("No rows found for delete query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(404)
		response := globals.Response{
			Status:  "error",
			Message: "No rows found matching the provided filters",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}
	defer rows.Close()

	// Get the deleteData
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

	var deleteData []map[string]interface{}
	var rowCount int
	var selectedAdminRows int
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
		deleteData = append(deleteData, rowData)
		log.Info("Selected row: " + rowData[globals.UserEntryIDColumnName].(string) + " from database: " + database + "/" + table + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		rowCount++
		log.Debug("User's (" + name + ") roles: " + fmt.Sprint(rowData["roles"]) + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		for index, column := range columns {
			if column == "roles" {
				if strings.Contains(fmt.Sprint(rowData[column]), globals.RolesSystemAdmin) {
					selectedAdminRows++
				}
			}
			log.Debug("Column " + fmt.Sprint(index) + ": " + column + " | Value: " + fmt.Sprint(rowData[column]))
		}

	}

	adminCountSelectQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE roles LIKE '%%%s%%'", table, globals.RolesSystemAdmin)
	log.Debug("Admin Count Select Query: " + adminCountSelectQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	adminCountRows, err := wrapper.Query(adminCountSelectQuery)
	if err != nil {
		log.Error("Failed to execute admin count select query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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
	defer adminCountRows.Close()

	var adminCount int
	for adminCountRows.Next() {
		if err := adminCountRows.Scan(&adminCount); err != nil {
			log.Error("Failed to scan row: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			continue
		}
	}

	log.Info("Admins marked for deletion: " + fmt.Sprint(selectedAdminRows) + "/" + fmt.Sprint(adminCount) + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	if selectedAdminRows == adminCount && selectedAdminRows > 0 {
		log.Warn("Attempted to delete all system admin users (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "This query was rejected because it would result in the deletion of all system admin users - Please restructure the query to ensure that at least one system admin user remains",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	if rowCount == 0 {
		log.Debug("No users found for deletion query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "No users found matching the provided filters",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	deleteQuery := "DELETE FROM " + table

	var delArgs []interface{}
	var argPlaceholders string
	var returnData []map[string]interface{}
	for _, row := range deleteData {
		if id, ok := row[globals.UserEntryIDColumnName].(string); ok {
			log.Debug("Row " + globals.UserEntryIDColumnName + ": " + row[globals.UserEntryIDColumnName].(string))
			delArgs = append(delArgs, id)
			argPlaceholders = argPlaceholders + ", ?"
			returnData = append(returnData, map[string]interface{}{globals.UserEntryIDColumnName: data.Process(id)})
		} else {
			// Debugging: Print a warning if the ID is not found or not a string
			log.Warn("ID not found or not a string in row: ", row)
		}
	}

	deleteQuery += " WHERE " + globals.UserEntryIDColumnName + " IN (" + strings.TrimPrefix(argPlaceholders, ",") + ")"

	log.Debug("Delete Query: " + deleteQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	// Execute the delete query
	_, err = wrapper.Execute(deleteQuery, userID, delArgs...)
	if err != nil {
		log.Error("Failed to execute delete query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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

	type responseDataStruct struct {
		AffectedRows int                      `json:"affectedRows"`
		IDs          []map[string]interface{} `json:"ids"`
	}

	responseData := responseDataStruct{
		AffectedRows: rowCount,
		IDs:          returnData,
	}

	// Send the deleteData back as a JSON response
	response := globals.Response{
		Status:  "success",
		Message: "User(s) deleted",
		Data:    responseData,
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
	log.Info("Successfully deleted " + fmt.Sprint(rowCount) + " user(s) from database: " + database + "/" + table + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
}
