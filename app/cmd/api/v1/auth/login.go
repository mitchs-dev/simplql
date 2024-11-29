package auth

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchs-dev/library-go/generator"
	jwtLib "github.com/mitchs-dev/library-go/jwt"
	"github.com/mitchs-dev/library-go/networking"
	authPkg "github.com/mitchs-dev/simplQL/pkg/api/auth"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	log "github.com/sirupsen/logrus"
)

func Login(r *http.Request, w http.ResponseWriter, userID, correlationID string) {
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
		log.Error("Login is not needed when JWT is disabled (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "Login is not needed when JWT is disabled on this server",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	database := r.URL.Query().Get("database")

	name, password, jwt, err := authPkg.AuthenticationHeaderData(r.Header.Get(globals.AuthenticationAuthorizationHeader), correlationID)
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

	if password == "" && name == "" && jwt == "" {
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

	var (
		userExists bool
	)

	if name != "" && password != "" {
		// Check if the user exists
		userExists, userID, _, err = authPkg.CheckBasic(name, password, database)
		if err != nil {
			if strings.Contains(err.Error(), globals.ErrorNotExist) {
				log.Error("User does not exist (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
				w.WriteHeader(404)
				response := globals.Response{
					Status:  "error",
					Message: "User (" + userID + ") does not exist",
					Data:    map[string]string{"correlationID": correlationID},
				}
				err := json.NewEncoder(w).Encode(response)
				if err != nil {
					log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
				}
				return
			}
			log.Error("Failed to check if user exists: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			w.WriteHeader(500)
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
		userExists, userID, name, _, err = authPkg.CheckJWT(jwt, database)
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
			w.WriteHeader(500)
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
		w.WriteHeader(400)
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
		w.WriteHeader(404)
		log.Error("User does not exist: " + userID + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		response := globals.Response{
			Status:  "error",
			Message: "User " + name + " (" + userID + ") does not exist",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response" + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		}
		return
	}

	log.Debug("User " + name + " (" + name + ") exists (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	// Generate JWT
	subject := "JWT Token for " + name + " for use in " + database
	data := generator.RandomString(globals.JWTRandomDataLength)
	jwt, _, jwttimeout, err := jwtLib.GenerateToken(authPkg.GetJWTSigningKey(database), globals.JWTTimeZone, globals.JWTTimeoutPeriod, authPkg.SetJWTIssuer(database), subject, userID, data)
	if err != nil {
		log.Error("Failed to generate JWT", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		w.WriteHeader(500)
		response := globals.Response{
			Status:  "error",
			Message: "Failed to generate JWT",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	// Commit JWT to database
	err = commitJWT(userID, jwt, jwttimeout, database)
	if err != nil {
		log.Error("Failed to commit JWT to database: ", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		w.WriteHeader(500)
		response := globals.Response{
			Status:  "error",
			Message: "Failed to commit JWT to database",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}
	log.Debug("JWT generated for user (" + name + ") (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	w.Header().Set(globals.AuthenticationHeaderJWTSessionToken, jwt)
	w.Header().Set(globals.AuthenticationHeaderSessionTimeout, fmt.Sprint(jwttimeout))

	response := globals.Response{
		Status:  "success",
		Message: "User " + name + " (" + userID + ") authenticated",
		Data:    map[string]string{"correlationID": correlationID, "jwt": jwt, "timeout": fmt.Sprint(jwttimeout)},
	}
	err = json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
	}
	log.Info("User " + name + " (" + userID + ") logged in to database: " + database + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
}
