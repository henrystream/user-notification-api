Develop a high-performance, scalable API using Fiber that includes user authentication, session management, WebSockets, background job processing, and rate limiting. The system should handle high traffic and be optimized for concurrency.

Task Description

You will build a User Management & Notification System that supports authentication, real-time messaging via WebSockets, role-based access control, background job processing, and API rate limiting.

Requirements

1. User Authentication & Session Management
	•	Register (POST /register)
	•	Accepts email, password, and role (admin/user).
	•	Passwords must be hashed (bcrypt) before storage.
	•	Login (POST /login)
	•	Authenticates user, generates a JWT token, and starts a session.
	•	Logout (POST /logout)
	•	Invalidates the session and token.
	•	Get Profile (GET /profile)
	•	Returns user info (only accessible if authenticated).

2. Role-Based Access Control (RBAC)
	•	Admin-only route (GET /admin)
	•	Accessible only if role = admin.
	•	User-only routes (GET /user-data)
	•	Regular users cannot access admin endpoints.

3. Middleware Implementation
	•	JWT Authentication Middleware
	•	Ensures protected routes are accessible only with a valid token.
	•	Rate Limiting Middleware
	•	Limits the number of requests per user (e.g., 100 requests per minute).
	•	Request Logging Middleware
	•	Logs request details (method, path, response time, and user ID).

4. WebSockets for Real-time Notifications
	•	Implement a WebSocket server (/ws) that sends real-time notifications to authenticated users.
	•	Users receive a “Welcome back” message upon login via WebSocket.
	•	Implement a global chat feature, allowing users to send messages to connected users.

5. Background Job Processing (Worker Queue)
	•	Implement a background worker (e.g., using NATS, RabbitMQ, or Kafka) to process:
	•	Email notifications when a new user registers.
	•	Asynchronous log processing (store logs in a DB).
	•	Message queuing for sending WebSocket notifications without blocking the main server.

6. API Monitoring & Performance Optimization
	•	Implement Prometheus metrics for tracking request latency and API usage.
	•	Add structured logging with Zap or Logrus.
	•	Optimize for high concurrency using Go routines effectively.

Bonus Points
	•	Implement OAuth2 (Google/GitHub login).
	•	Add a Two-Factor Authentication (2FA) system.
	•	Deploy as Dockerized microservices with Kubernetes orchestration.
	•	Implement GraphQL support for flexible data fetching.
	•	Implement gRPC for inter-service communication.

Expected Deliverables
	•	README.md with:
	•	Compilation and execution instructions.
	•	Config file format explanation.
	•	API usage details.
