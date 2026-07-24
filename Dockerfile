
# Dev-only image for the FMU backend.
# Hot-reloads via Air: any change under ./cmd or ./internal triggers a rebuild
# and the running binary is restarted in-place.
FROM golang:1.26-alpine AS build

# Tools Air shells out to during a rebuild
RUN apk add --no-cache git

# Install Air (newer air-verse/air fork)
RUN go install github.com/air-verse/air@latest

WORKDIR /app

# Pre-fetch deps so the first build doesn't hit the network
COPY go.mod go.sum ./
RUN go mod download

# Source is bind-mounted at runtime; this COPY only seeds the image
COPY . .

RUN go build -o /tmp/fmu cmd/api/main.go

FROM alpine:3.24 AS prod

WORKDIR /app

RUN mkdir -p internal/db/migrations 
COPY ./internal/db/migrations/ ./internal/db/migrations/
COPY --from=build /tmp/fmu /app/fmu

CMD ["/app/fmu"]