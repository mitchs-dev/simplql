package system

import (
	"encoding/json"
	"net/http"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/version"
	"github.com/mitchs-dev/library-go/networking"
	log "github.com/sirupsen/logrus"
)

func Version(r *http.Request, w http.ResponseWriter, correlationID string) {
	log.Debug("Version requested (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	symantic, hash := version.Read()
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
