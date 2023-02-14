# Websocket Server Clustering

A specific connection may exist on any server. Consider the possibility of 100's of
instances of the server, federation between servers would be very inefficient. Therefore, ID's should
be stored in a central location (Cassandra?) and updated by the server managing the connection.
Each server will know it's own hostname within the cluster, so the identity of and websocket
will be a combination of the server's name along with a random ID assigned by that server.

Let us consider the typical use-case scenario: a user sends a message to a chat room.

* Client connection sends an encrypted binary message to the server.
* Server persists the message in Postgres database. This MUST SUCCEED before proceeding. The
  message will not be un-encrypted anywhere besides the client's device, so any fields which need to
  be validated must be sent separately from the message.
* Server looks up list of clients currently subscribed to the specific chat room.
    * This list should be stored in a central location, i.e. Cassandra data store.
    * The server which manages the connection should be responsible for adding/removing
      subscribers for each chat room.
    * Websockets should be closed after a certain period of time to help load balance connections.
      If a server happens to crash on an instance, the subscriber ID can safely be removed after
      this TTL.
* Server sends message (GRPC?) directly to each server which is managing a connection to a subscriber
    * If a message cannot be sent to a subscriber, we will not notify of any errors. If a user
      has disconnected, they may have refreshed (re-initialized websocket) or closed the client.
      If the client re-connects, they will automatically refresh all messages which have
      been sent since the last update time, which is stored locally.
