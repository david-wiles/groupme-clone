# REST

The REST api, located in cmd/courier-rest, provides the entrypoint for any actions from clients. When a client
wishes to post a new message, create a room, update their profile, or do other activities which originate from the 
client, the request will be done as an HTTPS request to this service. 

## New Message

In the case of a new message or post, the message will be sent to all other clients subscribed to that room through the 
courier service. This service will do a series of tasks to persist the data and transmit to any connected clients:

1. Parse the binary message (GRPC service)
2. Persist the message bytes in RDMS
3. Query the clients currently subscribed to a room
4. Send messages to courier service for each client

## 