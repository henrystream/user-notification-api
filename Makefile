up:
	docker-compose up -d

down:
	docker-compose down

test:
	go test ./tests/... -v

run-app:
	go run main.go

run: up test run-app