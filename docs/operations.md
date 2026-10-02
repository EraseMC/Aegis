# Operations

## Integration

Wire version 2 requires a matching Practice-New plugin and Aegis build.
Keep TCP port 19140 private. The PHP bridge uses a nonblocking connection and
rebuilds sidecar state after disconnects; the game continues if Aegis is unavailable.

Keep `ERASE_AEGIS_PUNISH=false` during validation. Reach_A, KillAura_A, Speed_A
and Fly_A additionally default to action `none` in the sidecar configuration.
Violation thresholds are evidence for review, not proof of cheating.

## Detection limits

- Reach_A: repeated attacks beyond 3.35 blocks from input position to confirmed
  target bounds. Uses acknowledgment history and interpolation envelopes.
- KillAura_A: repeated attacks missing recent look-direction rays. Touch input
  is excluded because tap-to-attack does not require crosshair aim.
- Speed_A: sustained horizontal movement outside a conservative speed envelope.
- Fly_A: sustained hovering or upward movement inconsistent with basic gravity.

These checks do not implement full movement prediction or wall obstruction checks.
They require fresh state and acknowledgments with RTT <= 500ms. Flight, special
terrain, low TPS, death and frozen movement are excluded. Knockback and teleports
receive a two-second grace period; nearby block changes receive a shorter grace.

## Resource limits

Per player: 128 targets, 16 history samples per target, 16 outstanding markers
and 256 pending observations. Overflow invalidates history.

The PHP tracker coalesces updates and samples at 4Hz, at most eight players per tick.
Each sample reads up to 12 cells from already-loaded chunks. Snapshot recovery is
spread over ticks; there is no world scan or full block-map mirror.

The bridge limits input volume, estimated work and nonblocking I/O. Its output
queue is capped at 512KiB and discarded if stale. Work estimates cap elapsed time
per operation at 100us to tolerate scheduling and GC pauses. These are soft budgets,
not hard real-time limits. Overload causes a reconnect instead of blocking the game.

Practice-New's Compose configuration limits Aegis to 0.5 CPU and 192MiB memory,
with `GOMEMLIMIT=144MiB`. Builds run Go tests and compilation with one worker.

Status logs every 30 seconds include online sessions, acknowledged sessions,
average RTT and input count. `synced` indicates acknowledgment, not check eligibility.

## Validation and deployment

BenchmarkEngine100Players models one complete 50ms tick for 100 players at 20 CPS,
with 4Hz state/acknowledgments and ten opponents per player. It measures processing
cost, not live server TPS, network traffic or PHP tracker overhead.

Before a wire upgrade, back up plugin source, config and environment, and tag the
old image. Build Aegis, stop Practice gracefully, recreate Aegis, then start Practice.
Rollback must restore both sides together. Do not replace live worlds or plugin data.
