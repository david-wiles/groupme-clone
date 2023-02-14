proto:
	protoc --go_out=paths=source_relative:./internal -I./protobuf \
    	--go-grpc_out=paths=source_relative:./internal -I./protobuf \
    	protobuf/courier.proto
