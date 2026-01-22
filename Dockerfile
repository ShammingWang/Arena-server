FROM golang:1.21-alpine AS build

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/arena ./server/cmd/server

FROM alpine:3.19
WORKDIR /app
COPY --from=build /bin/arena /app/arena
EXPOSE 8080
ENTRYPOINT ["/app/arena"]
