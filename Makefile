run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

test:
	go test ./... -v

lint:
	go vet ./...

# Docker Compose — full stack (api + postgres + redis + nginx/frontend)
up:
	docker compose up --build

up-detach:
	docker compose up --build -d

down:
	docker compose down

down-volumes:
	docker compose down -v

logs:
	docker compose logs -f

# Build the API image only
docker-build:
	docker build -t exchange-rate-api .
