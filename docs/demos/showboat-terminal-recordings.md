# Showboat Terminal Recordings

Showboat remains the source of truth for executable markdown demos in this repository. Terminal recordings are a companion artifact, not a replacement for `showboat verify`.

## Artifact Convention

- Keep the markdown demo in `demos/<name>.md`.
- Keep the terminal recording next to it in `demos/<name>.cast`.
- When a demo needs a repeatable capture script, store it in `demos/<name>-recording.sh`.
- Reference the cast from the markdown demo using a relative link so the artifact is still discoverable in plain markdown renderers.

## Recording Helper

Use `demos/record-terminal.sh` to generate casts with repo-local defaults:

- headless Asciinema capture
- overwrite enabled for repeatable regeneration
- `--return` so command failures fail the recording run
- prompt-style command echoing for readability
- default terminal size `100x28`
- default idle cap `1.25` seconds

Record a single command:

```bash
./demos/record-terminal.sh \
  --output demos/example.cast \
  --title "Example terminal walkthrough" \
  --command 'showboat verify demos/zoekt-search-demo.md'
```

Record a scripted multi-step flow:

```bash
./demos/record-terminal.sh \
  --output demos/zoekt-search-demo.cast \
  --title "Zoekt demo terminal walkthrough" \
  --script demos/zoekt-search-demo-recording.sh
```

## Script Mode

Scripts sourced by `record-terminal.sh` can call these helpers:

- `run "cmd"`: print `$ cmd`, then execute it
- `note "text"`: print a plain note line
- `pause <seconds>`: add visual pacing between steps

Use script mode when the recording needs multiple commands or deliberate pauses for readability.

## Verification Boundary

Keep the two verification steps separate:

1. Verify the markdown demo with `showboat verify demos/<name>.md`
2. Verify the cast is parseable with `asciinema convert -f txt demos/<name>.cast -`

Do not fold cast playback into `showboat verify`. The markdown output should stay deterministic even when the recording timeline changes.

## Future Expansion

If a later demo needs reliable key-by-key automation for an interactive TUI, add an advanced `tmux`-driven mode on top of this helper. `tmux` is not required for the current Showboat companion-cast workflow.
