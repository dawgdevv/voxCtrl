<!-- SPECKIT START -->
For additional context about technologies to be used, project structure,
shell commands, and other important information, read the current plan
<!-- SPECKIT END -->

# AGENTS.md — voxctrl

## Build & Test

```bash
# Build the daemon binary
go build -o voxctrl ./cmd/voxctrl

# Run all unit tests (pure Go, no hardware needed)
go test ./...

# Full integration test — checks audio devices, whisper binaries, input permissions
bash scripts/test.sh
```

## Architecture

Single Go binary (`cmd/voxctrl/main.go`) that wires five pipeline stages via Go channels:

```
Hotkey Listener → Audio Capture → STT (Whisper) → Intent Parser → Command Executor
                                      ↓                                    ↓
                              whisper-server (HTTP)               Session Logger (SQLite)
```

- **Hotkey**: evdev (`/dev/input`), Ctrl+Alt hardcoded (hold both to activate, release either to stop). No X11/Wayland dependency.
- **Audio**: arecord (ALSA), 16kHz mono WAV to `/tmp/voxctrl_input.wav`.
- **STT**: prefers resident `whisper-server` on `:8080`, falls back to `whisper-cli`.
- **Intent**: fuzzy Levenshtein match against `config/commands.yaml`.
- **Executor**: shell commands via `bash -c`, notifies via system tray.
- **Session log**: SQLite at `~/.local/share/voxctrl/sessions.db`.

## Runtime Requirements

- **Linux only**. User must be in `input` group to read `/dev/input/event*`.
- **Build dependency**: `libayatana-appindicator-glib-dev` (CGO required by `fyne.io/systray`).
- **Runtime binaries**: `arecord`, `whisper-server` (preferred) or `whisper-cli`.
- **Model default**: `/usr/local/share/whisper/ggml-base.en.bin`.

## Environment Variables

| Variable | Default | Purpose |
|---|---|---|
| `VOXCTRL_DEVICE` | auto-detect | `/dev/input/eventX` for hotkey |
| `VOXCTRL_ALSA_DEVICE` | `default` | ALSA capture device |
| `VOXCTRL_MODEL` | `/usr/local/share/whisper/ggml-base.en.bin` | Whisper model path |
| `VOXCTRL_WHISPER_THREADS` | `runtime.NumCPU()` | Whisper threads |
| `VOXCTRL_WHISPER_BEAM` | `1` | Whisper beam size |
| `VOXCTRL_GPU_LAYERS` | `0` | GPU offload layers |

## Key Conventions

- **Command registry**: `config/commands.yaml` is hot-reloaded within ~2 seconds. No daemon restart needed.
- **Hotkey change**: edit `handleEvent` in `internal/hotkey/listener.go` to change the modifier chord or add a trigger key.
- **Systemd service**: `deploy/voxctrl.service` is a user unit. Install to `~/.config/systemd/user/`.
- **Logs**: daemon writes to stdout/stderr and `~/.local/share/voxctrl/voxctrl.log`. View live with `journalctl --user -u voxctrl -f`.
- **Binary artifact**: `voxctrl` binary is gitignored but often present in the working tree for local testing.

## Testing Notes

- Unit tests do **not** require hardware, whisper binaries, or ALSA devices. They test logic in isolation.
- `internal/hotkey` tests simulate kernel events with fake `inputEvent` structs.
- `internal/stt` tests mock HTTP responses and file I/O.
- `internal/session` tests use `t.TempDir()` and `t.Setenv("HOME", ...)` to avoid touching real `~/.local/share`.
- `scripts/test.sh` is the only test path that exercises real audio, whisper, and input devices.
