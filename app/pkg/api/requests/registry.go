/*
This package is used to store the functions which are used by the requests made via API calls from the requests package.
*/
package requests

import (
	"encoding/json"
	"net/http"

	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/simplQL/cmd/api/v1/auth"
	"github.com/mitchs-dev/simplQL/cmd/api/v1/db"
	"github.com/mitchs-dev/simplQL/cmd/api/v1/docs"
	"github.com/mitchs-dev/simplQL/cmd/api/v1/system"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/globals"
	log "github.com/sirupsen/logrus"
)

/*
	FUNCTION REGISTRY

IMPORTANT: You MUST map any functions you wish to use to keys here to be able to call them via the requests package
The key should be the name of the category and action of the request, separated by a hyphen (I.e. "category-action")

IMPORTANT: Do not forget to import the package containing the functions you wish to use
*/
var functionRegistry = map[string]requestHandlingFunction{
	// V1
	"auth-create":    auth.Create,
	"auth-read":      auth.Read,
	"auth-update":    auth.Update,
	"auth-delete":    auth.Delete,
	"auth-login":     auth.Login,
	"auth-logout":    auth.Logout,
	"db-create":      db.Create,
	"db-read":        db.Read,
	"db-update":      db.Update,
	"db-delete":      db.Delete,
	"docs-api":       docs.API,
	"system-version": system.Version,
	"system-healthz": system.Healthz,
}

// requestHandlingFunction is the function signature for the request handling functions - It requires a request, response writer, User ID, and a correlation ID as input and returns an error
type requestHandlingFunction func(*http.Request, http.ResponseWriter, string, string)

// Get returns a request handling function from the registry
func Get(name string) (requestHandlingFunction, bool) {
	function, exists := functionRegistry[name]
	return function, exists
}

// RunFunction runs a function from the registry
func RunFunction(functionName string, r *http.Request, w http.ResponseWriter, userID, correlationID string) {
	function, exists := Get(functionName)
	if exists {
		log.Debug("Calling function: " + functionName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + " U: " + userID + ")")
		function(r, w, userID, correlationID)
	} else {
		log.Warn("Function not found: " + functionName + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
		w.WriteHeader(500)
		response := globals.Response{
			Status:  "error",
			Message: "INTERNAL_SERVER_ERROR",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Error encoding response", err)
		}
	}

}
