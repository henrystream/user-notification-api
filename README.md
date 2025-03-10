# User Notification API

A high-performance API built with Fiber (Go) featuring user authentication, OAuth2 (Google), 2FA (TOTP), WebSockets, background jobs, and rate limiting. Uses PostgreSQL for data storage and Redis for sessions/rate limiting.

The user-notification-api is a Go-based service providing user authentication, real-time WebSocket chat, and notification capabilities. It integrates with Postgres for persistent storage, Redis for session management, Kafka for event-driven notifications, and gRPC for inter-service communication. Built with Fiber for HTTP/WebSocket handling, it supports JWT-based authentication and rate-limiting.

## Expected Deliverables

This README.md covers:

Compilation and Execution Instructions: Steps to build and run locally or in Docker/Kubernetes.
Config File Format Explanation: Details on configuration via environment variables.
API Usage Details: Endpoints, request/response formats, and examples.

## Features

Authentication: Register, login, and 2FA verification with JWT tokens.
WebSocket Chat: Real-time global chat for authenticated users.
Notifications: Email notifications via Kafka consumer.
Metrics: Prometheus metrics exposed at /metrics.
gRPC: Notification service endpoint at port 50051.
Rate Limiting: 100 requests per minute per IP.
Logging: Structured request logging.

## Prerequisites

- Go: 1.21 or later
- Docker: For containerized deployment
- Kubernetes: For cluster deployment (e.g., Minikube for local testing)

# Tools:

- curl for API testing
- wscat for WebSocket testing (npm install -g wscat)

## Compilation and Execution Instructions

Local Development

1. Clone the Repository:
   git clone https://github.com/henrystream/user-notification-api.git
   cd user-notification-api

2. Install Dependencies:

go mod download

3. Run Dependencies with Docker:

docker run -d --name postgres -p 5432:5432 -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password123 -e POSTGRES_DB=userdb postgres:latest
docker run -d --name redis -p 6379:6379 redis:latest
docker run -d --name zookeeper -p 2181:2181 -e ALLOW_ANONYMOUS_LOGIN=yes bitnami/zookeeper:latest
docker run -d --name kafka -p 9092:9092 -e KAFKA_BROKER_ID=1 -e KAFKA_ZOOKEEPER_CONNECT=host.docker.internal:2181 -e KAFKA_ADVERTISED_LISTENERS=PLAINTEXT://localhost:9092 -e KAFKA_LISTENERS=PLAINTEXT://0.0.0.0:9092 -e KAFKA_CREATE_TOPICS=user-registration:1:1 bitnami/kafka:latest

4. Set Environment Variables:

export POSTGRES_HOST=localhost
export POSTGRES_USER=postgres
export POSTGRES_PASSWORD=password123
export POSTGRES_DB=userdb
export KAFKA_BROKER=localhost:9092
export REDIS_HOST=localhost:6379

5. Compile and Run:

go build -o user-notification-api main.go
./user-notification-api

Alternatively: go run main.go
OR
make run-app

## Docker Deployment

1. Build the Image:

docker build -t user-notification-api:latest .

2. Run with Dependencies:

docker run -p 3000:3000 -p 50051:50051 -e KAFKA_BROKER=host.docker.internal:9092 -e POSTGRES_HOST=host.docker.internal -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password123 -e POSTGRES_DB=userdb -e REDIS_HOST=host.docker.internal:6379 user-notification-api:latest

3. Push to Registry:

docker tag user-notification-api:latest yourusername/user-notification-api:latest
docker push yourusername/user-notification-api:latest

## Kubernetes Deployment

1. Start Minikube:

minikube start --cpus=4 --memory=4096

2. Apply Manifests:

kubectl apply -f kubernetes/

3. Verify Pods:

kubectl get pods

4. Access Service:

minikube tunnel
minikube service app-service --url

## Config File Format Explanation

The application uses environment variables for configuration (no standalone config file). These are injected via ConfigMap in Kubernetes or set manually/Docker.

Environment Variables
Variable Description Default Required
POSTGRES_HOST Postgres host localhost Yes
POSTGRES_USER Postgres user postgres Yes
POSTGRES_PASSWORD Postgres password password123 Yes
POSTGRES_DB Postgres database userdb Yes
KAFKA_BROKER Kafka broker address localhost:9092 Yes
REDIS_HOST Redis host localhost:6379 Yes

# Example Config (Docker)

docker run -e POSTGRES_HOST=postgres -e POSTGRES_USER=postgres -e POSTGRES_PASSWORD=password123 -e POSTGRES_DB=userdb -e KAFKA_BROKER=kafka:9092 -e REDIS_HOST=redis:6379 user-notification-api:latest

# Kubernetes Config

Defined in kubernetes/configmap.yaml:

## API Usage Details

Base URL
Local: http://localhost:3000
Kubernetes: http://<minikube-ip>:3000 (from minikube service app-service --url)

# Endpoints

1.  Register User
    Method: POST
    Path: /register
    Request:
    {
    "email": "tes12@example.com",
    "password": "password123",
    "role": "user"
    }

    Response (200 OK):
    {
    "totp_secret": "some_secret_string"
    }

    Errors:
    400: {"error": "Invalid input"}
    500: {"error": "Database error"}

2.  Login
    Method: POST
    Path: /login
    Request:
    {
    "email": "tes12@example.com",
    "password": "password123"
    }
    Response (200 OK):
    {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    }

    Errors:

    401: {"error": "Invalid credentials"}
    500: {"error": "Database not available"}

3.  WebSocket Chat
    Method: WebSocket (GET with upgrade)
    Path: /ws
    Query Param: token=<jwt-from-login>
    URL Example: ws://localhost:3000/ws?token=eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
    Message Format (Send):
    {"message": "Hello"}
    Response Format (Receive):
    {"user_id": 1, "message": "Hello"}
    Errors:
    401: {"error": "Unauthorized"} (if token is missing/invalid)

4.  Metrics

    Method: GET
    Path: /metrics
    Response: Prometheus text format

    login_attempts_total{status="success"} 5
    websocket_connections_active 2

## Usage Examples

1. Register
   curl -X POST -H "Content-Type: application/json" -d '{"email":"tes12@example.com","password":"password123","role":"user"}' http://localhost:3000/register

2. Login
   curl -X POST -H "Content-Type: application/json" -d '{"email":"tes12@example.com","password":"password123"}' http://localhost:3000/login

3. Websocket
   wscat -c "ws://localhost:3000/ws?token=<token>"
   # Type: {"message":"Hello"}

## Troubleshooting

Pods Pending: Check kubectl describe pod <pod-name> for scheduling or image issues.
Connection Errors: Run minikube tunnel for external access.
API Fails: Verify logs with kubectl logs -l app=user-notification-api.

## Project structure

user-notification-api/
├── handlers/ # HTTP, WebSocket, and gRPC handlers
├── middleware/ # Fiber middleware (JWT, logging, rate-limiting)
├── models/ # Data models (e.g., User)
├── services/ # Database, Redis, Kafka initialization and logic
├── proto/ # gRPC protobuf definitions
├── kubernetes/ # Kubernetes manifests
├── Dockerfile # Docker image definition
├── go.mod # Go module dependencies
├── main.go # Application entry point
└── README.md # This file
