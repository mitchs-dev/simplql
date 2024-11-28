package auth

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	"github.com/mitchs-dev/library-go/networking"
	log "github.com/sirupsen/logrus"
)

func Update(r *http.Request, w http.ResponseWriter, correlationID string) {
	c.GetConfig()

	table := globals.UsersTable

	// Get the request body
	requestBody, err := io.ReadAll(r.Body)
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

	// Decode the request body into the EntryUpdate struct
	arb.getAuthRequest(requestBody, correlationID)

	log.Info("Using database: " + arb.Database + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	dbFilePath := c.Storage.Path + "/" + arb.Database + ".db"
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		log.Fatal("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()

	var filters string
	var filtersAsList []string
	var filterArgs []interface{}
	var args []interface{}

	if arb.Data.Update == nil {
		log.Error("Invalid request body - Update field is required (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request body - Update field is required",
			Data:    map[string]string{"correlationID": correlationID},
		}
		w.WriteHeader(http.StatusBadRequest)
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	log.Debug("Data to be updated: ", arb.Data.Update)
	updateData := arb.Data.Update
	var setClauses []string
	for field, value := range updateData {
		if field == "roles" {
			if !validateSystemRoles(value) {
				log.Error("Invalid role format or role (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				response := globals.Response{
					Status:  "error",
					Message: "Invalid role or role format - Ensure that the role format is valid (Ex: " + globals.SystemRolePrefix + "<role>) and the role is one of: " + globals.RolesSystemAdmin + ", " + globals.RolesSystemUser + ", " + globals.RolesSystemReadOnly,
					Data:    map[string]string{"correlationID": correlationID},
				}
				w.WriteHeader(http.StatusBadRequest)
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
		}
		setClauses = append(setClauses, field+" = ?")
		args = append(args, value)
	}

	if arb.Data.ID != "" {
		filtersAsList = append(filtersAsList, "id = ?")
		filterArgs = append(filterArgs, arb.Data.ID)
	}
	if arb.Data.Name != "" {
		filtersAsList = append(filtersAsList, "name = ?")
		filterArgs = append(filterArgs, arb.Data.Name)
	}
	if arb.Data.Password != "" {
		filtersAsList = append(filtersAsList, "password = ?")
		filterArgs = append(filterArgs, arb.Data.Password)
	}
	if len(arb.Data.Roles) > 0 {
		var rolesList []string
		for _, role := range arb.Data.Roles {
			rolesList = append(rolesList, "roles LIKE ?")
			if !strings.Contains(role, "%") {
				if strings.HasPrefix(role, "*") || strings.HasSuffix(role, "*") {
					role = strings.TrimSuffix(strings.TrimPrefix(role, "*"), "*")
					filterArgs = append(filterArgs, "%"+role+"%")
				} else {
					filterArgs = append(filterArgs, "%"+role+"%")
				}
			} else {
				filterArgs = append(filterArgs, role)
			}
		}
		filtersAsList = append(filtersAsList, strings.Join(rolesList, " AND "))
	}
	filters = strings.Join(filtersAsList, " AND ")
	args = append(args, filterArgs...)

	selectQuery := "SELECT roles," + globals.UserEntryIDColumnName + " FROM " + table + " WHERE " + filters
	log.Debug("Select query to check for selected admins: " + selectQuery)
	log.Debug("Select query args: ", filterArgs)

	// Execute the select query
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

	var selectedAdminRowsInt64 int64
	for rows.Next() {
		// Check if the user is a system admin
		var roles string
		var id string
		if err := rows.Scan(&roles, &id); err != nil {
			log.Error("Failed to scan row: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			continue
		}
		if strings.Contains(roles, globals.RolesSystemAdmin) {
			selectedAdminRowsInt64++
		}
	}
	if err := rows.Err(); err != nil {
		log.Error("Error iterating over rows: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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

	log.Info("Rows affected: ", selectedAdminRowsInt64)
	selectedAdminRows := int(selectedAdminRowsInt64)

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

	log.Info("Admins marked for update: " + fmt.Sprint(selectedAdminRows) + "/" + fmt.Sprint(adminCount) + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	if adminCount != 0 {
		if selectedAdminRows == adminCount {
			log.Warn("Attempted to update all system admin users (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "This query was rejected because it would result in the update of all system admin users - Please restructure the query to ensure that it does not affect all system admin users",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
	}

	updateQuery := "UPDATE " + table + " SET " + strings.Join(setClauses, ", ") + " WHERE " + filters
	log.Debug("Update query: ", updateQuery)
	log.Debug("Update query args: ", args)

	result, err := wrapper.Execute(updateQuery, args...)
	if err != nil {
		if strings.Contains(err.Error(), globals.ErrorTransactionNoEntry) {
			log.Error("No entry found for update query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "ENTRY_NOT_FOUND",
				Data:    map[string]string{"correlationID": correlationID},
			}
			w.WriteHeader(http.StatusNotFound)
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
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

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		log.Error("Failed to retrieve affected rows: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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

	response := globals.Response{
		Status:  "success",
		Message: "ENTRY_UPDATED",
		Data: map[string]interface{}{
			"correlationID": correlationID,
			"rowsAffected":  rowsAffected,
		},
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
}
