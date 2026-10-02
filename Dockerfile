FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN GOMAXPROCS=1 go test -p 1 ./... && CGO_ENABLED=0 GOMAXPROCS=1 go build -p 1 -trimpath -ldflags "-s -w" -o /aegis ./cmd/aegis

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /data
COPY --from=build /aegis /usr/local/bin/aegis
EXPOSE 19140/tcp
ENTRYPOINT ["/usr/local/bin/aegis", "-config", "/data/aegis.json"]
