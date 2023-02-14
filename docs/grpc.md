# Websocket Server GRPC Connections

Each websocket server has a GRPC endpoint which is only used internally for communication between servers. This 
communication is required to enable horizontal scaling across multiple servers.

Crate: tonic (grpc), prost, prost-types (protobuf)
