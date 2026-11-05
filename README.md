# gophkeeper

protoc \
  -I pkg/proto \
  --go_out=internal/proto \
  --go_opt=paths=source_relative \
  --go-grpc_out=internal/proto \
  --go-grpc_opt=paths=source_relative \
  pkg/proto/gophkeeper.proto

  dsn := "postgres://gophuser:user@localhost:5432/gophkeeper?sslmode=disable"

mkdir -p certs cmd/client && openssl req -x509 -newkey rsa:4096 -nodes -keyout certs/ca.key -out cmd/client/ca.crt -days 365 -subj "/CN=GophKeeperRootCA" && openssl req -newkey rsa:2048 -nodes -keyout certs/server.key -out certs/server.csr -subj "/CN=localhost" && echo "subjectAltName=DNS:localhost,IP:127.0.0.1" > certs/server.ext && openssl x509 -req -in certs/server.csr -CA cmd/client/ca.crt -CAkey certs/ca.key -CAcreateserial -out certs/server.crt -days 365 -extfile certs/server.ext && rm -f certs/server.csr certs/server.ext cmd/client/ca.srl cmd/client/ca.key && mv -f ca.srl certs/ 2>/dev/null || true

 "jwt_secret_token"