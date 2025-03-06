# User Notification API

A high-performance API built with Fiber (Go) featuring user authentication, OAuth2 (Google), 2FA (TOTP), WebSockets, background jobs, and rate limiting. Uses PostgreSQL for data storage and Redis for sessions/rate limiting.

## Features

- **User Authentication**: Register, login, logout (JWT-based).
- **OAuth2**: Google login integration.
- **Two-Factor Authentication (2FA)**: TOTP via authenticator apps.
- **Rate Limiting**: 100 requests/minute per user/IP.
- **WebSockets**: Real-time notifications and chat.
- **Background Jobs**: Redis-based (extendable for email/logs).
- **Monitoring**: Prometheus metrics at `/metrics`.

## Prerequisites

- Go 1.23+
- Docker & Docker Compose
- PostgreSQL, Redis
- Google OAuth2 credentials

## Setup

1. **Clone the Repository**:
   ```bash
   git clone https://github.com/henrystream/user-notification-api.git
   cd user-notification-api
   ```
