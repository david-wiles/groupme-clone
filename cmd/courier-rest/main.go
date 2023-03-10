package main

import (
	"database/sql"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/julienschmidt/httprouter"
	_ "github.com/lib/pq"
	"github.com/redis/go-redis/v9"
	log "github.com/sirupsen/logrus"
	"net/http"
	"os"
	"time"
)

var (
	db                 *sql.DB
	accountQueryEngine internal.AccountQueryEngine
	roomQueryEngine    internal.RoomQueryEngine
	messageQueryEngine internal.MessageQueryEngine
	rdb                *redis.Client
	jwtSecret          []byte
	courierConns       *internal.CourierConns

	isDev = false
)

func init() {
	// Set up logrus
	log.SetOutput(os.Stdout)
	log.SetFormatter(&log.TextFormatter{
		FullTimestamp:   true,
		TimestampFormat: time.RFC3339Nano,
	})

	// Initiate postgres connection
	connection := internal.MustGetEnv("POSTGRES_URI")

	var err error
	db, err = sql.Open("postgres", connection)
	if err != nil {
		panic(err)
	}

	if err = db.Ping(); err != nil {
		panic(err)
	}

	accountQueryEngine = internal.AccountQueryEngine{db}
	roomQueryEngine = internal.RoomQueryEngine{db}
	messageQueryEngine = internal.MessageQueryEngine{db}

	// Set JWT secret
	secret := internal.MustGetEnv("JWT_SECRET")

	jwtSecret = []byte(secret)

	// Initiate Redis connection
	redisAddr := internal.MustGetEnv("REDIS_ADDR")

	rdb = redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	courierConns = internal.NewCourierConnCache(rdb)

	// Set if local development
	if _, ok := os.LookupEnv("DEV"); ok {
		isDev = true
	}
}

func devHandler(next http.Handler) http.Handler {
	if isDev {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Access-Control-Allow-Origin", "*")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

			if r.Method != http.MethodOptions {
				next.ServeHTTP(w, r)
			}
		})
	} else {
		return next
	}
}

func loggingHandler(router *httprouter.Router) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		loggedWriter := &LoggedResponseWriter{200, w}

		start := time.Now().Nanosecond()
		router.ServeHTTP(loggedWriter, r)
		end := time.Now().Nanosecond()

		log.WithFields(log.Fields{
			"status":         loggedWriter.statusCode,
			"user-agent":     r.Header.Get("User-Agent"),
			"path":           r.URL.Path,
			"host":           r.Header.Get("Host"),
			"method":         r.Method,
			"responseTimeMs": (end - start) / 1000000.0,
		}).Infoln()
	}
}

type LoggedResponseWriter struct {
	statusCode int
	base       http.ResponseWriter
}

func (w *LoggedResponseWriter) Header() http.Header             { return w.base.Header() }
func (w *LoggedResponseWriter) Write(bytes []byte) (int, error) { return w.base.Write(bytes) }
func (w *LoggedResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.base.WriteHeader(statusCode)
}

func main() {
	router := httprouter.New()

	apiVersion := internal.MustGetEnv("API_VERSION")

	AddAccountRoutes("/api/"+apiVersion, router)
	AddRoomRoutes("/api/"+apiVersion, router)
	AddMessageRoutes("/api/"+apiVersion, router)

	if err := http.ListenAndServe(":9000", devHandler(loggingHandler(router))); err != nil {
		panic(err)
	}
}
