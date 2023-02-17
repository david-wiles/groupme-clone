package main

import (
	"context"
	"flag"
	"fmt"
	"github.com/david-wiles/groupme-clone/internal"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/gorilla/websocket"
	"github.com/montanaflynn/stats"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"log"
	"math/rand"
	"sync"
	"time"
	"unsafe"
)

func CreateWebsocketClient(ctx context.Context, URL string) (*pkg.Client, string, error) {
	conn, _, err := websocket.DefaultDialer.Dial(URL, nil)
	if err != nil {
		return nil, "", err
	}

	// Get this connection's UUID
	if err := conn.WriteMessage(websocket.TextMessage, []byte("whoami")); err != nil {
		return nil, "", err
	}

	_, bytes, err := conn.ReadMessage()
	if err != nil {
		_ = conn.WriteMessage(websocket.CloseMessage, websocket.FormatCloseMessage(websocket.CloseNormalClosure, ""))
		_ = conn.Close()
		return nil, "", err
	}

	client := pkg.NewClient(ctx, conn)

	whoami := pkg.WhoAmIResponse{}
	if err := client.Deserialize(bytes, &whoami); err != nil {
		client.Close()
	}

	return client, whoami.UUID, nil
}

func CreateGRPCClient(URL string) (internal.CourierClient, error) {
	conn, err := grpc.Dial(URL, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, err
	}

	client := internal.NewCourierClient(conn)
	return client, nil
}

const letterBytes = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
const (
	letterIdxBits = 6                    // 6 bits to represent a letter index
	letterIdxMask = 1<<letterIdxBits - 1 // All 1-bits, as many as letterIdxBits
	letterIdxMax  = 63 / letterIdxBits   // # of letter indices fitting in 63 bits
)

var src = rand.NewSource(time.Now().UnixNano())

// RandStringBytesMaskImprSrcUnsafe gets a random string of length n. Code from
// https://stackoverflow.com/questions/22892120/how-to-generate-a-random-string-of-a-fixed-length-in-go
func RandStringBytesMaskImprSrcUnsafe(n int) string {
	b := make([]byte, n)
	// A src.Int63() generates 63 random bits, enough for letterIdxMax characters!
	for i, cache, remain := n-1, src.Int63(), letterIdxMax; i >= 0; {
		if remain == 0 {
			cache, remain = src.Int63(), letterIdxMax
		}
		if idx := int(cache & letterIdxMask); idx < len(letterBytes) {
			b[i] = letterBytes[idx]
			i--
		}
		cache >>= letterIdxBits
		remain--
	}

	return *(*string)(unsafe.Pointer(&b))
}

func GetResults(data []float64) (float64, float64, float64, float64, float64, float64, error) {
	min, err := stats.Min(data)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	avg, err := stats.Percentile(data, 50)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	median, err := stats.Median(data)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	p90, err := stats.Percentile(data, 90)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	p95, err := stats.Percentile(data, 95)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}
	max, err := stats.Max(data)
	if err != nil {
		return 0, 0, 0, 0, 0, 0, err
	}

	return min, avg, median, p90, p95, max, nil
}

type Results struct {
	grpcErrors    int
	grpcDuration  []float64
	expectedCount int
	actualCount   int
}

func (r *Results) Merge(other *Results) {
	r.grpcErrors += other.grpcErrors
	r.grpcDuration = append(r.grpcDuration, other.grpcDuration...)
	r.expectedCount += other.expectedCount
	r.actualCount += other.actualCount
}

func (r *Results) Print() {
	min, avg, median, p90, p95, max, _ := GetResults(r.grpcDuration)
	const millisInNano = 1000000.00

	_, _ = fmt.Printf(`
######### Test complete ######### 

GRPC Errors: %d
GRPC Latency - min: %.2fms avg: %.2fms med %.2fms p90: %.2fms p95: %.2fms max: %.2fms
Expected Count: %d
Actual Count: %d
`, r.grpcErrors, min/millisInNano, avg/millisInNano, median/millisInNano, p90/millisInNano, p95/millisInNano, max/millisInNano, r.expectedCount, r.actualCount)
}

func StartVirtualUser(wsURL, grpcURL string, interval time.Duration, gracefulStop int, logInfo bool, ctx context.Context) (Results, error) {
	var (
		nextPayload string
		mu          = sync.Mutex{}

		grpcErrors    = 0
		grpcDuration  []float64
		expectedCount = 0
		actualCount   = 0
	)

	if logInfo {
		log.Printf("opening new websocket url=%s\n", wsURL)
	}
	websocketClient, id, err := CreateWebsocketClient(ctx, wsURL)
	if err != nil {
		return Results{}, err
	}

	if logInfo {
		log.Printf("opening connection to GRPC server url=%s\n", grpcURL)
	}
	grpcClient, err := CreateGRPCClient(grpcURL)
	if err != nil {
		return Results{}, err
	}

	// Read all messages
	go func() {
		for msg := range websocketClient.Reads {
			if message, ok := msg.(*pkg.ClientMessage); ok {
				if logInfo {
					log.Printf("websocket received payload=%s cid=%s acknowledge=%t uuid=%s\n", message.Payload, message.Cid, message.Acknowledge, id)
				}
				// We expect this message to equal `nextPayload`
				mu.Lock()
				if string(message.Payload) == nextPayload {
					actualCount++
				}
				mu.Unlock()
			}
		}
	}()

	// Send a message to the websocketClient at a set interval
	ticker := time.NewTicker(interval)
	for {
		select {
		case <-ticker.C:

			mu.Lock()
			nextPayload = RandStringBytesMaskImprSrcUnsafe(128)
			if logInfo {
				log.Printf("sending message payload=%s uuid=%s\n", nextPayload, id)
			}
			start := time.Now()
			_, err := grpcClient.SendMessage(context.Background(), &internal.MessageRequest{
				Uuid:    id,
				Payload: []byte(nextPayload),
			})
			end := time.Now()
			mu.Unlock()

			grpcDuration = append(grpcDuration, float64(end.Sub(start).Nanoseconds()))

			if err != nil {
				grpcErrors++
				log.Printf("grpc error err=%v", err)
			} else {
				expectedCount++
			}
		case <-ctx.Done():
			// Graceful stop.. don't overload the server with disconnections.
			time.Sleep(time.Millisecond * 1000 * time.Duration(rand.Intn(gracefulStop)))
			if logInfo {
				log.Printf("closing connection uuid=%s\n", id)
			}
			websocketClient.Close()
			return Results{grpcErrors, grpcDuration, expectedCount, actualCount}, nil
		}
	}
}

// client-tester is a load tester based on k6, but avoids the limitation of the global event loop. The test
// will create virtual users with ramp-up and a limit of users, and repeats the test until the test is complete.
// Each user reports its own results which the test combines into a final report.
//
// First, a connection to the websocket server and grpc server is initialized. The grpc client will then continuously
// initiate messages to send to the websocket through courier. The websocket reads the messages and records the results
// to ensure the messages match. It additionally records latency and any grpc errors.
//
// Due to the nature of the event loop, once the test duration is complete any ongoing tests will not be completed
func main() {
	wsURL := flag.String("ws-url", "ws://localhost:8080", "The URL of the chat server")
	grpcURL := flag.String("grpc-url", "localhost:8081", "The URL for the GRPC server")
	interval := flag.Duration("interval", time.Second, "Interval between messages to the websocketClient")
	duration := flag.Duration("duration", 10*time.Minute, "Total test duration")
	userDuration := flag.Duration("user-duration", time.Minute, "Duration for a single user")
	users := flag.Int("users", 10, "Number of users to run during the test")
	jitter := flag.Int("jitter", 500, "Max jitter between user ramp-up. This is the max value, the actual jitter"+
		"will be between 0 and this value.")
	gracefulStop := flag.Int("graceful-stop", 10, "Jitter value for graceful stop of websocket clients. Actual value "+
		"will be between 0 and this value in seconds")
	logInfo := flag.Bool("log-info", false, "Log info messages")

	flag.Parse()

	ctx, cancel := context.WithTimeout(context.Background(), *duration)
	defer cancel()

	allResults := &Results{}
	resultChan := make(chan Results)

	// Read results and merge once a test completes
	go func() {
		for result := range resultChan {
			allResults.Merge(&result)
		}
	}()

	// Use a wait group to ensure all tests complete
	wg := sync.WaitGroup{}

	log.Printf("Starting test. Websocket URL: %s GRPC URL: %s, Message interval: %v, Graceful stop jitter: %ds", *wsURL, *grpcURL, *interval, *gracefulStop)

	for i := 0; i < *users; i++ {
		// Add jitter between user creation
		time.Sleep(time.Millisecond * time.Duration(rand.Intn(*jitter)))
		wg.Add(1)

		// Create user in separate thread. This will continuously create a new user once the previous one exits
		// Once the total test is done, this will exit and the client will clean up its own resources
		go func() {
			// Keep creating users until the parent context (total test time) is cancelled
			for ctx.Err() == nil {
				wsCtx, cancel := context.WithTimeout(ctx, *userDuration)

				// All resources from StartVirtualUser are released once out of scope. This includes the
				// graceful stop of websocket connections
				results, err := StartVirtualUser(*wsURL, *grpcURL, *interval, *gracefulStop, *logInfo, wsCtx)
				if err != nil {
					panic(err)
				}

				resultChan <- results
				cancel()
			}
			wg.Done()
		}()
	}

	wg.Wait()

	allResults.Print()
}
