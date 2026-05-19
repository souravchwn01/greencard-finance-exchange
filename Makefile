run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v

docker-build:
	docker build -t exchange-rate-api .

lint:
	go vet ./...
