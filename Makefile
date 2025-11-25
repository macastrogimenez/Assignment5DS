.PHONY: proto clean run-node1 run-node2 run-node3 run-client

proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		proto/auction.proto

clean:
	rm -f proto/*.pb.go
	rm -f node/node
	rm -f client/client

build: proto
	go build -o node/node ./node
	go build -o client/client ./client

run-node1:
	go run node/main.go -id=1 -port=5001 -peers=localhost:5002,localhost:5003

run-node2:
	go run node/main.go -id=2 -port=5002 -peers=localhost:5001,localhost:5003

run-node3:
	go run node/main.go -id=3 -port=5003 -peers=localhost:5001,localhost:5002

run-client:
	go run client/main.go -nodes=localhost:5001,localhost:5002,localhost:5003
