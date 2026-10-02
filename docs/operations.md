# Operations

## Integration

Wire version 2 requires a matching Practice-New plugin and Aegis build.
Keep TCP port 19140 private. The PHP bridge uses a nonblocking connection and
rebuilds sidecar state after disconnects; the game continues if Aegis is unavailable.

Keep `ERASE_AEGIS_PUNISH=false` during validation.
Violation thresholds are evidence for review, not proof of cheating.

With enforcement enabled, the default policy is:

| Checks | Action |
| --- | --- |
| Reach_A, KillAura_A, Timer_A | Ban for 14 days |
| Fly_A, Speed_A, BadPacket_A, Autoclicker_A | Kick |

Ban duration is enforced by the Practice bridge, not encoded in the sidecar action.
Other integrations must define their own duration. BadPacket_A has a threshold of
1 VL; other checks use 10 VL. Movement and combat also require repeated evidence
before increasing VL. No single reach sample causes a ban.

## Detection limits

- Reach_A: repeated attacks with nearest hitbox distance above 3.05 blocks, or
  non-touch ray distance above 3.01 after a 0.1-block hitbox expansion. Uses
  acknowledgment history, interpolation envelopes and adjacent attacker positions.
  Attacks are evaluated on the following input to accommodate packet ordering.
- KillAura_A: repeated attacks missing recent look-direction rays. Touch input
  is excluded because tap-to-attack does not require crosshair aim.
- Speed_A: sustained horizontal movement outside a conservative speed envelope.
- Fly_A: consecutive single-tick vertical changes inconsistent with gravity.
  Jump impulses reset evidence even if a ground sample was missed. Gaps reset
  prediction; ceilings and partial collision shapes are excluded.

These checks do not implement full movement prediction or wall obstruction checks.
They require fresh state and acknowledgments with RTT <= 500ms. Exact timestamp
unit conversions (x1000/x1000000) are matched against outstanding markers only.
Flight, low TPS, death and frozen movement are excluded. Movement additionally
excludes special terrain and receives a two-second knockback grace; nearby block
changes receive a shorter grace. Combat remains active during knockback. Teleports
reset history and give both check groups a two-second grace.

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
combat/movement eligibility, average RTT and input count. `synced` indicates
acknowledgment; `combat_ready` and `movement_ready` indicate current eligibility.

## Validation and deployment

BenchmarkEngine100Players models one complete 50ms tick for 100 players at 20 CPS,
with 4Hz state/acknowledgments and ten opponents per player. It measures processing
cost, not live server TPS, network traffic or PHP tracker overhead.

Before a wire upgrade, back up plugin source, config and environment, and tag the
old image. Build Aegis, stop Practice gracefully, recreate Aegis, then start Practice.
Rollback must restore both sides together. Do not replace live worlds or plugin data.
