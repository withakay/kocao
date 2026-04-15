# Showboat Terminal Recording PoC

Date: `2026-04-15`

Artifacts:
- Cast: `../../demos/showboat-terminal-recording-poc.cast`
- Plain-text export: `showboat-terminal-recording-poc.txt`

Goal:
- Validate that a repo-local Asciinema cast can live next to Showboat demos without changing `showboat verify`.

Workflow Recorded:
- `showboat init demo.md "Asciinema PoC Demo"`
- `showboat note demo.md "Record a small showboat workflow."`
- `showboat exec demo.md bash "echo ready"`
- `showboat verify demo.md`
- `showboat extract demo.md`

Recording Command:

```bash
asciinema record --headless --overwrite --return --idle-time-limit 1.25 \
  --window-size 100x28 \
  --title "Showboat terminal recording PoC" \
  --command "<scripted shell workflow>" \
  demos/showboat-terminal-recording-poc.cast
```

Findings:
- The cast was generated successfully with `asciinema 3.2.0` using a non-interactive `--command` flow.
- The artifact is small: `demos/showboat-terminal-recording-poc.cast` is about `1.1K`.
- `asciinema convert -f txt demos/showboat-terminal-recording-poc.cast docs/agents/showboat-terminal-recording-poc.txt --overwrite` succeeded, so parseability can be used as a non-interactive verification step.
- A sibling cast in `demos/` feels workable for the eventual checked-in artifact convention.
- The playback/readability problem is not output capture, it is command visibility: scripted headless recordings do not show typed input, so the script had to print `$ <command>` lines explicitly.
- Compared against the external `terminal-recording` skill pattern, `tmux` looks like the right optional automation layer if we later need reliable key delivery and screen synchronization for true interactive TUI demos.
- That same comparison also argues against making `tmux`, `zellij`, or `agg` required for v1: the current Kocao use case is checked-in `.cast` companions for Showboat demos, not GIF rendering or complex storyboard playback.

Recommendation:
- Proceed with the proposal as a Kocao-specific variation on that skill, but bake command echoing into the helper from the start.
- Keep cast verification separate from `showboat verify`.
- Use relative markdown links to the `.cast` file rather than relying on a custom embedded player.
- If later demos need synchronized navigation through a real TUI, add an advanced `tmux` mode rather than introducing `zellij` as a dependency.
