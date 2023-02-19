# Courier

Courier is the name of the service responsible for pushing new messages to active clients. 

This service may be deployed across many nodes and serves as a stateless proxy for communicating with any websockets. 
The specific URI of any client can be obtained through a centralized Cassandra data store. The URI should be used with
courier's GRPC service to send messages. The GRPC server and websocket server exist on the same process so that the 
GRPC call will directly send a message to the client.
