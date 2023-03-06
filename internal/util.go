package internal

import (
	"encoding/json"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
)

func MustGetEnv(key string) string {
	val, ok := os.LookupEnv(key)
	if !ok {
		panic("no environment variable found for " + key)
	}
	return val
}

func SerializeResponse(w http.ResponseWriter, r any) {
	encoder := json.NewEncoder(w)
	if err := encoder.Encode(r); err != nil {
		log.WithFields(log.Fields{
			"err": err,
		}).Errorln("unable to serialize response")
		w.WriteHeader(500)
	}
}
