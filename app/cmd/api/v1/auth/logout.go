package auth

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/library-go/networking"
	log "github.com/sirupsen/logrus"
)

func Logout(r *http.Request, w http.ResponseWriter, correlationID string) {
	c.GetConfig()

	if !c.Session.JWT.Enabled {
		if strings.HasPrefix(r.Header.Get(globals.AuthenticationAuthorizationHeader), globals.AuthenticationAuthorizationHeaderBearerPrefix) {
			log.Error("JWT is disabled on this server, but a JWT was provided in the Authorization header (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			response := globals.Response{
				Status:  "error",
				Message: "JWT is disabled on this server, but a JWT was provided in the Authorization header",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		log.Error("Logout is not needed when JWT is disabled (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Logout is not needed when JWT is disabled on this server",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	database := r.URL.Query().Get("database")

	name, password, jwt, err := authenticationHeaderData(r.Header.Get(globals.AuthenticationAuthorizationHeader), correlationID)
	if err != nil {
		log.Error("Failed to get authentication header data: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid Authorization header",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	if name == "" && password == "" && jwt == "" {
		log.Error("Failed to get authentication header data: ", "No data was provided in the Authorization header (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid Authorization header - Ensure that you are using Basic or Bearer authentication",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	if database == "" {
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request body - Database name is required",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	var (
		userExists bool
		userID     string
	)
	if name != "" && password != "" {
		// Check if the user exists
		userExists, userID, _, err = checkBasic(name, password, database)
		if err != nil {
			if strings.Contains(err.Error(), globals.ErrorNotExist) {
				log.Error("User name (" + name + ") does not exist (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				w.WriteHeader(404)
				response := globals.Response{
					Status:  "error",
					Message: "User name (" + name + ") does not exist",
					Data:    map[string]string{"correlationID": correlationID},
				}
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
			log.Error("Failed to check if user exists: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			response := globals.Response{
				Status:  "error",
				Message: "Failed to check if user exists",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
	} else if jwt != "" {
		log.Debug("JWT provided, checking if user exists")
		userExists, userID, name, _, err = checkJWT(jwt, database)
		if err != nil {
			if strings.Contains(err.Error(), globals.ErrorNotExist) {
				log.Error("JWT not found (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				w.WriteHeader(404)
				response := globals.Response{
					Status:  "error",
					Message: "JWT not found",
					Data:    map[string]string{"correlationID": correlationID},
				}
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
			log.Error("Failed to check if user exists: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			response := globals.Response{
				Status:  "error",
				Message: "Failed to check if user exists",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
	} else {
		log.Error("Invalid request body - Password, Name, or JWT is required (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request body - Password, Name, or JWT is required",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	if !userExists {
		log.Error("User does not exist: " + userID + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "User " + name + " (" + userID + ") does not exist",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	log.Debug("User " + name + " (" + userID + ") exists (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	err = deleteJWT(userID, database)
	if err != nil {
		if strings.Contains(err.Error(), globals.ErrorNotExist) {
			log.Warn("JWT does not exist for user " + name + " (" + userID + ") - User is considered already logged out (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			w.WriteHeader(200)
			response := globals.Response{
				Status:  "error",
				Message: "User " + name + " (" + userID + ") is already logged out",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		log.Error("Failed to delete JWT: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		response := globals.Response{
			Status:  "error",
			Message: "Failed to delete JWT for user (" + userID + ")",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	w.WriteHeader(200)
	response := globals.Response{
		Status:  "success",
		Message: "User " + name + " (" + userID + ") logged out successfully",
		Data:    map[string]string{"correlationID": correlationID},
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
	log.Info("User " + name + " (" + userID + ") logged out of database: " + database + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
}
