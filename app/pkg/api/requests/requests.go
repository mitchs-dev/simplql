/*
The requests package is used to handle all incoming requests to the server.

It has a provided request schema file which is used to validate incoming requests and
route them to the registry package to call the appropriate function.
*/
package requests

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/api/auth"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/configuration"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"

	"github.com/gorilla/mux"
	"github.com/mitchs-dev/library-go/generator"
	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/library-go/processor"
	log "github.com/sirupsen/logrus"

	"github.com/mitchs-dev/library-go/requestSchemas"
)

// Variables which are pointers to various configuration structs
var (
	c  configuration.Configuration
	rs requestSchemas.Schema
)

//go:embed requestSchema.yaml
var requestSchemaFile embed.FS

func Startup() {
	log.Debug("Starting up requests package")
	rsData, err := requestSchemaFile.ReadFile(globals.RequestSchemaFileName)
	if err != nil {
		log.Fatal("Error reading request schema file: " + err.Error())
	}

	globals.RequestSchemaData = rsData

	if len(globals.RequestSchemaData) == 0 {
		log.Fatal("Request schema file is empty - Please check the file: " + globals.RequestSchemaFileName)
	}

	log.Debug("Request schema file read successfully")
}

func interceptor(w http.ResponseWriter, r *http.Request) {

	c.GetConfig()
	correlationID := generator.CorrelationID("Local")
	w.Header().Set(globals.NetworkingHeaderCorrelationID, correlationID)
	log.Debug("Endpoint Hit: " + r.URL.Path + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	// Check if the request path is valid (Must have API slug, API version, and endpoint)
	if len(r.URL.Path) < 3 {
		log.Debug("Invalid request path: " + r.URL.Path + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(404)
		response := globals.Response{
			Status:  "error",
			Message: "Invalid request path",
			Data:    nil,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	// Verify the request method is valid
	switch r.Method {
	case "GET", "POST", "PUT", "DELETE":
		break
	default:
		log.Debug("Invalid request method: " + r.Method + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(405)
		response := globals.Response{
			Status:  "error",
			Message: "Method not allowed",
			Data:    nil,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	if r.URL.Path == "/favicon.ico" {
		log.Debug("Returning favicon.ico (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		http.ServeFile(w, r, "assets/favicon.ico")
		w.WriteHeader(200)
		fmt.Fprintf(w, "")
		return
	}

	// Get the category from the URL
	category := strings.ToLower(r.URL.Path)
	log.Debug("Processing request with category: " + category + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

	// Validate the request
	invalidReason, categoryIndex, actionIndex, userID, jwtTokenValue, jwtTokenExpireTime, err := runRequestValidation(r, correlationID)

	// Set the JWT Token and Expire Time in the response headers
	w.Header().Set(globals.AuthenticationHeaderJWTSessionToken, jwtTokenValue)
	w.Header().Set(globals.AuthenticationHeaderSessionTimeout, jwtTokenExpireTime)

	if err != nil {
		if invalidReason == globals.ErrorAuthenticationNoRoles {
			log.Error("User does not have a required role: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			w.WriteHeader(403)
			response := globals.Response{
				Status:  "error",
				Message: "Forbidden: You do not have a required role to complete the request",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		} else if invalidReason == globals.ErrorAuthenticationUserNotFound {
			log.Error("User not found: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			w.WriteHeader(401)
			response := globals.Response{
				Status:  "error",
				Message: "Unauthorized: User not found",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		} else if invalidReason == globals.ErrorAuthenticationJWTExpired {
			log.Error("Expired JWT: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			w.WriteHeader(401)
			response := globals.Response{
				Status:  "error",
				Message: "Unauthorized: JWT Expired",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		} else if invalidReason == "INTERNAL_SERVER_ERROR ("+correlationID+")" {
			log.Error("Internal server error: " + err.Error() + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
			w.WriteHeader(500)
			response := globals.Response{
				Status:  "error",
				Message: "INTERNAL_SERVER_ERROR",
				Data:    map[string]string{"correlationID": correlationID},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		if strings.Contains(invalidReason, "See server logs for details") {
			w.WriteHeader(400)
			response := globals.Response{
				Status:  "error",
				Message: "BAD_REQUEST",
				Data:    map[string]string{"correlationID": correlationID, "reason": invalidReason},
			}
			err := json.NewEncoder(w).Encode(response)
			if err != nil {
				log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
			}
			return
		}
		log.Warn(err.Error() + ": " + invalidReason + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(400)
		response := globals.Response{
			Status:  "error",
			Message: "BAD_REQUEST",
			Data:    map[string]string{"correlationID": correlationID, "reason": invalidReason},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Error encoding response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+networking.GetRequestIPAddress(r)+")")
		}
		return
	}

	log.Debug("Valid request: " + category + " (index: " + fmt.Sprint(categoryIndex) + "/" + fmt.Sprint(actionIndex) + ") (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + " U: " + userID + ")")

	rs.GetSchema(globals.RequestSchemaData)

	//// Get the category and action from the request schema
	category = rs.RequestSchema.Categories[categoryIndex].Name
	action := rs.RequestSchema.Categories[categoryIndex].Actions[actionIndex].Name
	// Combine the category and action to get make the functionRegistry key
	functionToCall := category + "-" + action
	// Finally, call the function from the registry
	RunFunction(functionToCall, r, w, userID, correlationID)

}

// ENDPOINT REQUEST HANDLERS

func Handler() {
	// Get Configuration
	c.GetConfig()

	// Set Router

	myRouter := mux.NewRouter().StrictSlash(true)

	// Root Endpoints
	myRouter.PathPrefix("/").HandlerFunc(interceptor)

	// Set listener for the backend
	listenAddress := c.Network.ListenAddress
	port := c.Network.Port
	ApplicationName := globals.ApplicationName
	if c.Network.TLS.Enabled {
		// Check that the TLS cert and key are set
		if c.Network.TLS.Cert == "" || c.Network.TLS.Key == "" {
			log.Fatal("TLS Cert and Key paths are required for HTTPS")
		}
		// Make sure that the cert and key paths are valid
		if !processor.DirectoryOrFileExists(c.Network.TLS.Cert) || !processor.DirectoryOrFileExists(c.Network.TLS.Key) {
			log.Fatal("TLS Cert and Key paths are invalid or do not exist")
		}
		log.Info(ApplicationName + " listening on port: " + fmt.Sprint(port))
		log.Info("Using protocol: HTTPS")
		err := http.ListenAndServeTLS(listenAddress+":"+fmt.Sprint(port), c.Network.TLS.Cert, c.Network.TLS.Key, myRouter)
		if err != nil {
			log.Fatal("Error in listening and serving ", err)
		}
	} else {
		log.Info(ApplicationName + " listening on port: " + fmt.Sprint(port))
		log.Info("Using protocol: HTTP")
		err := http.ListenAndServe(listenAddress+":"+fmt.Sprint(port), myRouter)
		if err != nil {
			log.Error("Error in listening and serving", err)
		}
	}

}

/*

	Request Validation Functions

*/

// runRequestValidation validates the request and returns Error Reason, Category Index, Action Index, User ID,  JWT Token, JWT Token Expire Time, and Error
func runRequestValidation(r *http.Request, correlationID string) (string, int, int, string, string, string, error) {

	var (
		hasCategory  bool
		hasAction    bool
		errorMessage string
		userID       string
	)

	// Ensure that the request contains the api path
	if !strings.Contains(r.URL.Path, globals.NetworkingAPIEndpoint) {
		return "Invalid path. Must be of type: " + globals.NetworkingAPIEndpoint, -1, -1, "", "", "", fmt.Errorf("invalid request")
	}

	// Ensure the request contains a valid category and action
	if len(strings.Split(r.URL.Path, "/")) < 5 {
		return "Request path must contain a category and action", -1, -1, "", "", "", fmt.Errorf("invalid request path")
	}

	// Set the request category and action
	requestCategory := strings.Split(r.URL.Path, "/")[3]
	requestAction := strings.Split(r.URL.Path, "/")[4]

	// JWT Token and Expire Time
	var (
		jwtTokenValue      string
		jwtTokenExpireTime string
		// TODO: Uncomment with authentication implementation
		//authUsername       string
	)

	rs.GetSchema(globals.RequestSchemaData)

	// Check if the request contains a valid category
	for i, category := range rs.RequestSchema.Categories {
		if category.Name == requestCategory {
			hasCategory = true

			// Check if the request contains a valid action
			for j, action := range category.Actions {
				if action.Name == requestAction {

					hasAction = true

					// Check if the request method and body are valid
					if action.Method != r.Method {
						return "Method not allowed", i, j, "", "", "", fmt.Errorf("invalid request")
					}

					if action.Body && r.ContentLength == 0 {
						return "Body is required", i, j, "", "", "", fmt.Errorf("invalid request")
					} else if !action.Body && r.Body != nil {
						// If the content length is not 0, then the body is not allowed
						if r.ContentLength != 0 {
							return "Body is not allowed", i, j, "", "", "", fmt.Errorf("invalid request")
						}
					}

					// Check if the request contains valid parameters
					if len(action.Parameters) > 0 {
						// Check if the request contains valid parameters
						for _, parameter := range action.Parameters {
							if r.FormValue(parameter) == "" {
								return "Parameter not found: " + parameter, i, j, "", "", "", fmt.Errorf("invalid request")
							}
						}
					}

					// Ensure the required headers are set
					if len(action.Headers.Request) > 0 {
						for _, header := range action.Headers.Request {
							if r.Header.Get(header.Name) == "" && header.Required {
								return "Required header (" + header.Name + ") not found for request type: " + action.Method + " " + globals.NetworkingAPIEndpoint + "/" + category.Name + "/" + action.Name, i, j, "", "", "", fmt.Errorf("invalid request")
							}
						}
					}

					log.Debug("Action roles: ", action.Roles)
					// Verify the user making the request has a required role
					if len(action.Roles) > 0 {

						log.Debug("Request requires authentication (" + category.Name + "/" + action.Name + ") - Ensuring user is authenticated")

						authorizationHeader := r.Header.Get(globals.AuthenticationAuthorizationHeader)

						if authorizationHeader == "" {
							return "Authorization header not found - Request (" + action.Method + " " + globals.NetworkingAPIEndpoint + "/" + category.Name + "/" + action.Name + ") requires authorization", i, j, "", "", "", fmt.Errorf("invalid request")
						}

						var arb auth.AuthRequestBody
						requestBody, err := io.ReadAll(r.Body)
						if err != nil {
							return "failed to read request body", i, j, "", "", "", fmt.Errorf("invalid request")
						}

						// Reset the request body so it can be read again
						r.Body = io.NopCloser(bytes.NewBuffer(requestBody))
						arb.GetAuthRequest(requestBody, correlationID)

						var erb globals.EntryRequest
						// Read the request body into a buffer
						bodyBytes, err := io.ReadAll(r.Body)
						if err != nil {
							return "failed to read request body", i, j, "", "", "", fmt.Errorf("invalid request")
						}
						// Decode the buffer into requestBody
						err = json.NewDecoder(bytes.NewBuffer(bodyBytes)).Decode(&requestBody)
						if err != nil {
							log.Debug("Request doesn't seem like an entry request")
						}

						// Reset the request body so it can be read again later
						r.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

						var database string
						if r.URL.Query().Get("database") != "" {
							database = strings.TrimPrefix(r.URL.Query().Get("database"), "\"")
							database = strings.TrimSuffix(database, "\"")
						} else if arb.Database != "" {
							database = arb.Database
						} else if erb.Database != "" {
							database = erb.Database
						} else {
							return "Database could not be found but is required for authentication", i, j, "", "", "", fmt.Errorf("invalid request - database not found in request")
						}

						userID, err = auth.RunAuthChecks(authorizationHeader, database, correlationID, action.Roles)
						if err != nil {
							return err.Error(), i, j, "", "", "", err
						}
						log.Debug("User is authenticated: " + userID + " and will continue with request validation (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")

					} else {

						log.Debug("No authentication required for the request: " + action.Name + " (" + category.Name + ")")

					}

					return "", i, j, userID, jwtTokenValue, jwtTokenExpireTime, nil
				}
			}
		}
	}

	if !hasCategory {
		errorMessage = "Category not found"
	}
	if !hasAction {
		errorMessage = "Action not found"
	}

	return errorMessage, -1, -1, "", "", "", fmt.Errorf("invalid request")

}
