setup:
	docker run -d --name postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password123 -e POSTGRES_DB=userdb postgres:latest
	docker run -d --name kafka -p 9092:9092 -e KAFKA_BROKER_ID=1 -e KAFKA_ZOOKEEPER_CONNECT=host.docker.internal:2181 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 -e KAFKA_CREATE_TOPICS=user-registration:1:1 bitnami/kafka:latest
	docker run -d --name zookeeper -p 2181:2181 -e ALLOW_ANONYMOUS_LOGIN=yes bitnami/zookeeper:latest
	docker run -d --name redis -p 6379:6379 redis:latest
up:
	docker-compose up -d

down:
	docker-compose down

test:
	go test ./tests/... -v

run-app:
	go run main.go

run: up test run-app