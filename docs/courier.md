# Courier

Courier is the name of the service responsible for pushing new messages to active clients. 

This service is deployed across many nodes and serves as a stateless proxy for communicating with any websockets. The
specific uri of any websocket can be obtained through a centralized Cassandra data store