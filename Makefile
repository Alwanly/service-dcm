# ---- build, run, and test
run:
	go run ./cmd/$(service)

build:
	go build -o ./app ./cmd/$(service)

build-docker:
	docker build -f ./docker/$(service)/Dockerfile -t service-$(service):latest .

run-docker:
	docker-compose -f docker-compose.yml up

format:
	go fmt ./...

test:
	go test ./internal/... ./pkg/...

# ---- dependencies
tidy:
	go mod tidy

install:
	go mod download

.PHONY: run dev build bump lint format test tidy install

swagger-generate:
	swag init -g cmd/controller/main.go -o docs/controller
	swag init -g cmd/worker/main.go -o docs/worker

swagger-controller:
	swag init -g cmd/controller/main.go -o docs/controller

swagger-worker:
	swag init -g cmd/worker/main.go -o docs/worker