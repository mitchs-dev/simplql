package auth

import (
	"encoding/json"
	"io"
	"net/http"
	"reflect"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/database/sqlWrapper"
	"github.com/mitchs-dev/library-go/generator"
	"github.com/mitchs-dev/library-go/networking"
	log "github.com/sirupsen/logrus"
)

var arb authRequestBody

var c configuration.Configuration

func Create(r *http.Request, w http.ResponseWriter, correlationID string) {

	c.GetConfig()

	// Form the request body
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

	arb.getAuthRequest(authBody, correlationID)
	// If the request body is empty, return an error
	if reflect.DeepEqual(arb, reflect.Zero(reflect.TypeOf(arb)).Interface()) {
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request body - Ensure that you are sending a valid JSON or YAML object",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	database := arb.Database
	id := arb.Data.ID
	name := arb.Data.Name
	password := arb.Data.Password
	roles := arb.Data.Roles
	log.Debug("Using database: " + database + " for query (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	if id != "" {
		log.Error("ID is automatically generated and prohibited on new user creation (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "ID is automatically generated and prohibited on new user creation",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	} else {
		id = generator.RandomString(globals.UserIDLength)
	}
	if name == "" {
		log.Error("Username (name) is required when creating a new user (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Username (name) is required when creating a new user",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	} else if name == c.Session.Default.Name {
		log.Error("Name is reserved and cannot be used (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Name is reserved and cannot be used",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}
	if len(roles) == 0 {
		log.Error("Roles are required when creating a new user (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Roles are required when creating a new user",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	} else {
		for _, role := range roles {
			if !strings.Contains(role, ":") {
				log.Error("Query contains invalid role format: " + role + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				response := globals.Response{
					Status:  "error",
					Message: "Invalid role format - Ensure that the role format is valid (Ex: " + globals.SystemRolePrefix + ":" + role + ")",
					Data:    map[string]string{"correlationID": correlationID},
				}
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
		}
	}
	if password == "" {
		log.Debug("Password is not provided - Generating a random password (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		password = generator.RandomString(globals.UserPasswordLength)
	}
	dbFilePath := c.Storage.Path + "/" + database + ".db"
	wrapper, err := sqlWrapper.NewSQLiteWrapper(dbFilePath)
	if err != nil {
		log.Fatal("Error when creating database wrapper: " + err.Error())
	}
	defer wrapper.Close()
	// Make sure that the user does not already exist
	selectQuery := "SELECT id FROM " + globals.UsersTable + " WHERE name = ?"
	log.Debug("Select Query: " + selectQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	rows, err := wrapper.Query(selectQuery, name)
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
		var id string
		if err := rows.Scan(&id); err != nil {
			log.Error("Failed to scan row: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			continue
		}
		log.Error("User already exists: " + name + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "User already exists: " + name,
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	log.Debug("Using roles: ", roles, " for user creation (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")

	// Convert roles slice to JSON string
	rolesJSON, err := json.Marshal(roles)
	if err != nil {
		log.Fatalf("Failed to marshal roles: %v", err)
	}

	log.Debug("Roles JSON: ", string(rolesJSON), " for user creation (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")

	// Create the user
	insertQuery := "INSERT INTO " + globals.UsersTable + " (id, name, password, roles) VALUES (?, ?, ?, ?)"
	log.Debug("Insert Query: " + insertQuery + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	_, err = wrapper.Execute(insertQuery, id, name, password, string(rolesJSON))
	if err != nil {
		log.Error("Failed to execute insert query: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
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
	log.Info("Created user: " + name + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	response := globals.Response{
		Status:  "success",
		Message: "User created",
		Data:    map[string]string{"correlationID": correlationID, "id": id, "name": name, "password": password, "roles": strings.Join(roles, ",")},
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}

}
