# User Notification API

A high-performance, scalable API built with Fiber (Go) featuring user authentication, session management, WebSockets, background job processing, and rate limiting. Uses PostgreSQL for data storage and Redis for sessions and rate limiting.

## Features

- **User Authentication**: Register (`/register`), Login (`/login`), Logout (via token expiration), Profile (`/profile`).
- **Role-Based Access Control**: Admin-only (`/admin`) and user-only (`/user-data`) routes.
- **Rate Limiting**: 100 requests per minute per user/IP.
- **WebSockets**: Real-time notifications (`/ws`) with welcome message and global chat.
- **Background Jobs**: Email notifications and log processing (via Redis, extendable).
- **Monitoring**: Prometheus metrics at `/metrics`.

## Prerequisites

- Go 1.23+
- Docker and Docker Compose

## Setup

1. **Clone the Repository**

   ```bash
   git clone <repository-url>
   cd user-notification-api
   ```

1. **Update Locally**:
   - Replace `C:/Users/henry/go/src/user-notification-api/README.md` with the above content.
   - Commit and push:
     ```bash
     git add README.md
     git commit -m "Update README with project details"
     git push origin main
     ```
