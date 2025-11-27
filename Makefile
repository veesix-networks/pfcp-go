.PHONY: proto
proto:
	protoc --go_out=. --go_opt=paths=source_relative \
		--go-grpc_out=. --go-grpc_opt=paths=source_relative \
		api/pfcp/v1/control.proto

.PHONY: build
build:
	go build -o bin/pfcp-cp ./cmd/pfcp-cp
	go build -o bin/pfcp-up ./cmd/pfcp-up

.PHONY: test
test:
	go test ./...

.PHONY: docker
docker:
	cd test/docker && docker compose up --build
