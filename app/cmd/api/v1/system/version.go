package system

import (
	"encoding/json"
	"net/http"
	"os"
	"strings"

	"github.com/mitchs-dev/library-go/networking"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/globals"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitialization/version"
	log "github.com/sirupsen/logrus"
)

func Version(r *http.Request, w http.ResponseWriter, userID, correlationID string) {
	log.Debug("Version requested (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	symantic, hash := version.Read()
	if strings.ToLower(os.Getenv(globals.GlobalDevelopmentBuildEnvironmentVariable)) == "true" {
		symantic = symantic + "-dev"
	}
	w.WriteHeader(200)
	response := globals.Response{
		Status:  "ok",
		Message: "Version information",
		Data: map[string]string{
			"symantic": symantic,
			"hash":     hash,
		},
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Error encoding response", err)
	}
}
