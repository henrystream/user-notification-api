FROM golang:1.23 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o user-notification-api

FROM golang:1.23
WORKDIR /app
COPY --from=builder /app/user-notification-api .
EXPOSE 3000 50051
CMD ["./user-notification-api"]

#FROM gcr.io/distroless/base
#WORKDIR /app
#COPY --from=builder /app/user-notification-api .
#EXPOSE 3000
#CMD ["./user-notification-api"]