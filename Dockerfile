FROM golang:1.26 AS build
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags "-s -w" -o /aegis ./cmd/aegis

FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /data
COPY --from=build /aegis /usr/local/bin/aegis
EXPOSE 19132/udp
ENTRYPOINT ["/usr/local/bin/aegis"]
