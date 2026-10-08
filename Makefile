IMAGE := googlehealth-mcp

.PHONY: run build test vet lint format docker-build docker-run docker-rm docker-logs

run:
	go run ./cmd/main

build:
	go build -o bin/server ./cmd/main

test:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run ./...

format:
	golangci-lint fmt ./...

docker-build:
	docker build -t $(IMAGE) .

docker-run: docker-build docker-rm
	docker run -d --name $(IMAGE) \
		--env-file .env \
		-e SERVER_HOST=0.0.0.0 \
		-e OAUTH_TOKEN_FILE=/data/token.json \
		-p 8080:8080 -p 8060:8060 \
		-v $(IMAGE)-data:/data \
		$(IMAGE)

docker-rm:
	docker rm -f $(IMAGE) 2>/dev/null || true

docker-logs:
	docker logs -f $(IMAGE)
