.PHONY: clean build build-server build-client build-all test cover proto cert generate

SERVER_NAME=server
CLIENT_NAME=client
BUILD_DIR=bin
VERSION=$$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
DATE=$$(date +'%Y-%m-%d_%H:%M:%S')
COMMIT=$$(git rev-parse --short HEAD 2>/dev/null || echo "unknown")
LDFLAGS=-ldflags "-X main.buildVersion=$(VERSION) -X main.buildDate=$(DATE) -X main.buildCommit=$(COMMIT) -s -w"

clean:
	@rm -rf $(BUILD_DIR)

build: build-server build-client

build-server:
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(SERVER_NAME) ./cmd/server/main.go

build-client:
	@mkdir -p $(BUILD_DIR)
	@go build $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME) ./cmd/client/main.go

build-all: clean
	@mkdir -p $(BUILD_DIR)
	@$(MAKE) build-server
	@GOOS=windows GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME)-windows-amd64.exe ./cmd/client/main.go
	@GOOS=linux GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME)-linux-amd64 ./cmd/client/main.go
	@GOOS=darwin GOARCH=amd64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME)-darwin-amd64 ./cmd/client/main.go
	@GOOS=darwin GOARCH=arm64 go build $(LDFLAGS) -o $(BUILD_DIR)/$(CLIENT_NAME)-darwin-arm64 ./cmd/client/main.go

test:
	@go test ./...

cover:
	@go test -coverprofile=coverage.out ./... && \
		grep -vE '\.pb\.go|/mocks/' coverage.out > filtered.out && \
		go tool cover -func=filtered.out
	@rm -f coverage.out filtered.out

proto:
	@mkdir -p internal/proto
	@protoc \
		-I pkg/proto \
		--go_out=internal/proto \
		--go_opt=paths=source_relative \
		--go-grpc_out=internal/proto \
		--go-grpc_opt=paths=source_relative \
		pkg/proto/gophkeeper.proto

cert:
	@mkdir -p certs cmd/client
	@printf "[req]\ndistinguished_name=dn\nprompt=no\n[dn]\nCN=GophKeeperRootCA\n[v3_ca]\nbasicConstraints=critical,CA:TRUE\nkeyUsage=critical,digitalSignature,keyCertSign\n" > certs/ca.cnf
	@openssl req -x509 -newkey rsa:4096 -nodes -keyout certs/ca.key -out cmd/client/ca.crt -days 365 -config certs/ca.cnf -extensions v3_ca
	@printf "[req]\ndistinguished_name=dn\nprompt=no\n[dn]\nCN=localhost\n" > certs/server.cnf
	@openssl req -newkey rsa:2048 -nodes -keyout certs/server.key -out certs/server.csr -config certs/server.cnf
	@printf "subjectAltName=DNS:localhost,IP:127.0.0.1\n" > certs/server.ext
	@openssl x509 -req -in certs/server.csr -CA cmd/client/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 365 -extfile certs/server.ext
	@rm -f certs/ca.cnf certs/server.cnf certs/server.csr certs/server.ext certs/ca.srl cmd/client/ca.srl cmd/client/ca.key ca.srl 2>/dev/null || true

generate:
	@mkdir -p docs
	@swag init -g cmd/server/main.go
	@$(MAKE) cert
	@$(MAKE) proto
	@go generate ./...
	@go mod tidy


	
