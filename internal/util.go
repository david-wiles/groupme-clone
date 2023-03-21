package internal

import (
	"encoding/json"
	"github.com/google/uuid"
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

func FilterUUID(list []uuid.UUID, toRemove uuid.UUID) []uuid.UUID {
	for i := 0; i < len(list); i++ {
		if list[i] == toRemove {
			list[i] = list[len(list)-1]
			return list[:len(list)-1]
		}
	}
	return list
}
