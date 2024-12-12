package docs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	yml "github.com/ghodss/yaml"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/globals"
	yaml "gopkg.in/yaml.v2"

	log "github.com/sirupsen/logrus"
)

// Get returns the documentation for the API.
func API(r *http.Request, w http.ResponseWriter, userID, correlationID string) {

	log.Debug("Retrieving API documentation (C: " + correlationID + " | M: " + r.Method + " | IP: " + r.RemoteAddr + ")")

	// We allow the user to specify the format of the API documentation if they wish
	format := strings.ToLower(r.URL.Query().Get("format"))

	// Will provide the final API documentation results

	var (
		rawDocs  []byte
		jsonData []byte
		err      error
	)

	// We want to default to JSON if the user does not specify a format
	if format == "" {
		format = "json"
	}

	switch format {

	case "yaml":

		/*
			Although the data is already in YAML format,
			we want to ensure that the data is formatted without any extra comments or whitespace.
		*/

		var yamlObj interface{}
		err := yaml.Unmarshal(globals.RequestSchemaData, &yamlObj)
		if err != nil {
			log.Printf("Error unmarshalling YAML data: %v", err)
			w.WriteHeader(500)
			fmt.Fprint(w, "{\"status\": \"error\", \"message\": \"INTERNAL_SERVER_ERROR ("+correlationID+")\"}")
			return
		}

		yamlData, err := yaml.Marshal(yamlObj)
		if err != nil {
			log.Printf("Error marshalling YAML data: %v", err)
			w.WriteHeader(500)
			fmt.Fprint(w, "{\"status\": \"error\", \"message\": \"INTERNAL_SERVER_ERROR ("+correlationID+")\"}")
			return
		}

		rawDocs = yamlData

	case "json":

		// Convert the YAML to JSON
		jsonData, err = yml.YAMLToJSON(globals.RequestSchemaData)
		if err != nil {
			log.Error("Error converting YAML to JSON:" + err.Error())
			w.WriteHeader(500)
			fmt.Fprint(w, "{\"status\": \"error\", \"message\": \"INTERNAL_SERVER_ERROR ("+correlationID+")\"}")
			return
		}

	default:

		// If the user specifies an invalid format, throw an error
		log.Warn("Invalid format specified for API documentation request: " + format + " (C: " + correlationID + " | M: " + r.Method + " | IP: " + r.RemoteAddr + ")")
		w.WriteHeader(400)
		response := globals.Response{
			Status:  "error",
			Message: "Invalid format specified for API documentation request",
			Data:    map[string]string{"correlationID": correlationID},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+r.RemoteAddr+")")
		}
		return

	}

	w.WriteHeader(200)
	if format == "json" {

		response := globals.Response{
			Status:  "success",
			Message: "API documentation retrieved successfully",
			Data:    map[string]interface{}{"correlationID": correlationID, "format": format, "documentation": json.RawMessage(jsonData)},
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+r.RemoteAddr+")")
		}
	} else {
		if len(rawDocs) == 0 {
			log.Error("API docs are empty")
			return
		}

		var parsedDocs interface{}
		err := yaml.Unmarshal([]byte(rawDocs), &parsedDocs)
		if err != nil {
			log.Error("Failed to unmarshal API docs", err.Error())
			return
		}

		response := globals.Response{
			Status:  "success",
			Message: "API documentation retrieved successfully",
			Data:    map[string]interface{}{"correlationID": correlationID, "format": format, "documentation": parsedDocs},
		}
		err = yaml.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Failed to encode response", err.Error()+" (C: "+correlationID+" | M: "+r.Method+" | IP: "+r.RemoteAddr+")")
		}
	}
}
