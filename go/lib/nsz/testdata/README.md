These fixtures pin the NCZ/NSZ byte layout used by upstream `nicoboss/nsz`.

Upstream does not commit public binary test fixtures; its CI references private
Switch dump directories. These small fixtures are deterministic, synthetic
homebrew-like byte streams encoded according to upstream `docs/formats.md`:

- first `0x4000` bytes are an NCA header rebuilt by tests
- `NCZSECTN` section table starts at `0x4000`
- solid fixtures continue with a zstd frame
- block fixtures continue with `NCZBLOCK` and independent zstd blocks

The zstd frames were generated with the system `zstd` CLI, not this package.
