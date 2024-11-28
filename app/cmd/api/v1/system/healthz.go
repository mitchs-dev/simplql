package system

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/mitchs-dev/simplQL/pkg/configurationAndInitalization/globals"
	"github.com/mitchs-dev/library-go/networking"
	log "github.com/sirupsen/logrus"
)

func Healthz(r *http.Request, w http.ResponseWriter, correlationID string) {
	healthZHeaderKey := r.Header.Get(globals.NetworkingHeaderHealthZ)
	if strings.ToLower(healthZHeaderKey) != "true" {
		w.WriteHeader(400)
		response := globals.Response{
			Status:  "error",
			Message: "INVALID_REQUEST: Healthz header (" + globals.NetworkingHeaderHealthZ + ") not set to true",
			Data:    nil,
		}
		err := json.NewEncoder(w).Encode(response)
		if err != nil {
			log.Error("Error encoding response", err)
		}
		return
	}

	log.Debug("Healthz requested (C: " + correlationID + " | M: " + r.Method + " | IP: " + networking.GetRequestIPAddress(r) + ")")
	w.WriteHeader(200)
	response := globals.Response{
		Status:  "ok",
		Message: "Alive",
		Data:    nil,
	}
	err := json.NewEncoder(w).Encode(response)
	if err != nil {
		log.Error("Error encoding response", err)
	}
}
