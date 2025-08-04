# Build stage
FROM golang:1.22-alpine AS build
WORKDIR /src
COPY go.mod .
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o portkey-server ./cmd/server

# Final stage
FROM scratch
COPY --from=build /src/portkey-server /portkey-server
EXPOSE 8080
ENTRYPOINT ["/portkey-server"]
