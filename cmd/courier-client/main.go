package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"github.com/david-wiles/groupme-clone/pkg"
	"github.com/gorilla/websocket"
	"io"
	"log"
	"net/http"
)

func createLoginRequestBody(username, password string) (io.Reader, error) {
	var loginBytes []byte
	req := &pkg.LoginRequest{username, password}
	buf := bytes.NewBuffer(loginBytes)
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(req); err != nil {
		return nil, err
	}
	return buf, nil
}

func createRegisterRequestBody(webhook string) (io.Reader, error) {
	var loginBytes []byte
	req := &pkg.ClientRegisterRequest{webhook}
	buf := bytes.NewBuffer(loginBytes)
	encoder := json.NewEncoder(buf)
	if err := encoder.Encode(req); err != nil {
		return nil, err
	}
	return buf, nil
}

func getJWTFromResponse(body io.ReadCloser) (string, error) {
	resp := &pkg.LoginResponse{}
	decoder := json.NewDecoder(body)
	if err := decoder.Decode(resp); err != nil {
		return "", err
	}

	if err := body.Close(); err != nil {
		return "", err
	}

	return resp.JWT, nil
}

// Creates a virtual device which will act as a chat client
func main() {

	wsURL := flag.String("ws-url", "ws://localhost:8080", "The URL of the Courier websocket server")
	restURL := flag.String("rest-url", "http://localhost:9000", "The URL of the Courier REST server")
	username := flag.String("username", "", "Your username")
	password := flag.String("password", "", "Your password")

	flag.Parse()

	body, err := createLoginRequestBody(*username, *password)
	if err != nil {
		panic(err)
	}

	resp, err := http.DefaultClient.Post(*restURL+"/account/login", "application/json", body)
	if err != nil {
		panic(err)
	}

	token, err := getJWTFromResponse(resp.Body)
	if err != nil {
		panic(err)
	}

	conn, _, err := websocket.DefaultDialer.Dial(*wsURL, nil)
	if err != nil {
		panic(err)
	}

	client, err := pkg.NewClient(context.Background(), conn)
	if err != nil {
		panic(err)
	}

	// Self-registration
	body, err = createRegisterRequestBody(client.Webhook)
	req, err := http.NewRequest("POST", *restURL+"/client/register", body)
	if err != nil {
		panic(err)
	}

	req.Header.Set("Authorization", "Bearer "+token)

	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		panic(err)
	}

	if resp.StatusCode != http.StatusOK {
		panic("client registration was unsuccessful")
	}

	for msg := range client.Reads {
		if message, ok := msg.(*pkg.ClientMessage); ok {
			log.Printf("%s", string(message.Payload))
		}
	}
}
