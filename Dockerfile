# CONTAINER FOR BUILDING BINARY
FROM golang:1.21 AS build

# INSTALL DEPENDENCIES
RUN go install github.com/gobuffalo/packr/v2/packr2@v2.8.3
COPY go.mod go.sum /src/
RUN cd /src && go mod download

# BUILD BINARY
COPY . /src
RUN cd /src/db && packr2
RUN cd /src && make build

# CONTAINER FOR RUNNING BINARY
FROM alpine:3.18 as poolmanager
COPY --from=build /src/dist/zkevm-pool-manager /app/zkevm-pool-manager
COPY --from=build /src/test/config/test.poolmanager.toml /app/poolmanager.toml
RUN apk update && apk add postgresql15-client
EXPOSE 8123
CMD ["/bin/sh", "-c", "/app/zkevm-pool-manager run --cfg /app/poolmanager.toml"]
