# --- Build stage ---
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o /portkey-server ./cmd/server

# --- Final stage ---
FROM scratch
COPY --from=build /portkey-server /portkey-server
EXPOSE 80 443 8080
ENTRYPOINT ["/portkey-server"]
