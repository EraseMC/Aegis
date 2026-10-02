# Aegis

Server-side anticheat for Minecraft Bedrock, written in Go.

Runs alongside PocketMine-MP. The [Practice-New](https://github.com/EraseMC/Practice-New)
plugin forwards player input and server observations over a private TCP connection.
Aegis returns violations and configured actions; the plugin handles alerts and enforcement.

## Checks

Timer, invalid packets, autoclicker, reach, aim consistency, speed and flight.
Autoclickers up to 20 CPS are allowed on all input devices.

Combat and movement checks are experimental and default to alert-only mode.
They require acknowledged client state and skip stale or lagged observations.
Movement detection uses conservative bounds, not a full physics simulation.

## Build

Requires Go 1.25.1 or later.

```sh
go build -o aegis ./cmd/aegis
cp aegis.example.json aegis.json
./aegis -config aegis.json
```

Listen address defaults to `:19140`. Do not expose this port publicly.
The plugin and sidecar must use the same wire version (currently `2`).

## Test

```sh
go test ./...
go vet ./...
go test ./internal/engine ./internal/player -run '^$' -bench . -benchmem
```

Deployment constraints and detector limitations: [operations](docs/operations.md).

## License

[MIT](LICENSE).
