# VoxCtrl

A local, always-on voice command daemon for Linux developers. Press a hotkey, speak a command, watch it execute. No cloud. No internet. No data leaves your machine.

Works on any Linux distribution — Ubuntu, Fedora, Arch, Debian, Manjaro, and more. Compatible with both X11 and Wayland.

---

## What It Does

VoxCtrl sits silently in the background as a system daemon. When you press `Alt+V` and speak, it records your voice, transcribes it locally using Whisper, matches it to a command, and runs it — all in under 1.5 seconds.

Every interaction is logged to a local SQLite database with full context: what you said, what ran, which window was active, which git branch you were on, and how long it took.

---

## How It Works

```
Hold Alt+V  →  speak  →  release  →  command executes
```

Internally, five subsystems chain together via Go channels:

```
Hotkey Listener → Audio Capture → Whisper STT → Intent Parser → Command Executor
                                                                        ↓
                                                               Session Logger (SQLite)
```

Each subsystem is a separate Go package with a clean interface. The hotkey listener reads directly from `/dev/input` via evdev — no dependency on X11 or any display server. This is what makes VoxCtrl work identically across all Linux environments.

---

## What You Can Say

**Developer Tools**
- `"open vscode"` — launches VSCode in the current directory
- `"open terminal"` — opens a new terminal window
- `"open browser"` — launches your default browser

**System**
- `"lock screen"` — locks the session
- `"volume up"` / `"volume down"` — adjusts system volume
- `"mute"` — toggles mute
- `"take screenshot"` — captures the screen

**Git**
- `"git status"` — shows short status as a desktop notification
- `"git pull"` — pulls latest from current branch
- `"what branch"` — notifies you of the current branch

**Docker**
- `"docker up"` — runs `docker-compose up -d` in the current directory
- `"docker down"` — runs `docker-compose down`
- `"docker status"` — shows running containers in a notification

Commands are fuzzy-matched, so `"hey open vs code"` and `"launch vscode"` both work. All commands are defined in `config/commands.yaml` — edit the file and changes take effect immediately without restarting the daemon.

---

## What Gets Logged

Every command creates a record in `~/.local/share/voxctrl/sessions.db`:

| Field | Example |
|---|---|
| Transcript | `"open vscode"` |
| Matched intent | `open vscode` |
| Match confidence | `0.94` |
| Result | `success` |
| Active window | `Terminal` |
| Git branch | `main` |
| Pipeline latency | `1243ms` |

---

## Resource Usage

Measured on a standard Linux machine with no GPU:

| State | Memory |
|---|---|
| Idle (model loaded, waiting) | ~188 MB total |
| Peak (during transcription) | ~316 MB for ~1.4 seconds |
| CPU at idle | near zero (event-driven) |

The Whisper model loads once at startup and stays resident. There is no per-command reload cost.

---

## Requirements

**System**

| Requirement | Purpose |
|---|---|
| Linux kernel 4.x+ | evdev input support |
| ALSA or PipeWire | Microphone capture |
| A notification daemon | Desktop notifications (any DE ships one) |
| User in `input` group | Read from `/dev/input` for hotkey detection |

**Tools**

| Tool | Install |
|---|---|
| Go 1.21+ | See below |
| whisper-cli | Build from [whisper.cpp](https://github.com/ggml-org/whisper.cpp) |
| arecord | `alsa-utils` package |
| notify-send | `libnotify` package |
| ffmpeg | `ffmpeg` package |

---

## Installation

**Step 1 — Install system dependencies**

```bash
# Debian / Ubuntu
sudo apt install -y golang-go alsa-utils libnotify-bin ffmpeg build-essential cmake git

# Fedora / RHEL
sudo dnf install -y golang alsa-utils libnotify ffmpeg cmake git gcc gcc-c++

# Arch / Manjaro
sudo pacman -S go alsa-utils libnotify ffmpeg cmake git base-devel

# openSUSE
sudo zypper install go alsa-utils libnotify-tools ffmpeg cmake git gcc
```

**Step 2 — Add yourself to the input group** (required for hotkey detection)

```bash
sudo usermod -aG input $USER
# Log out and back in for this to take effect
```

**Step 3 — Build and install whisper.cpp**

```bash
git clone https://github.com/ggml-org/whisper.cpp.git
cd whisper.cpp
cmake -B build && cmake --build build -j$(nproc) --config Release
bash models/download-ggml-model.sh base.en
sudo mkdir -p /usr/local/share/whisper
sudo cp ./build/bin/whisper-cli /usr/local/bin/whisper-cli
sudo cp models/ggml-base.en.bin /usr/local/share/whisper/ggml-base.en.bin
cd ..
```

**Step 4 — Build and install VoxCtrl**

```bash
git clone https://github.com/yourusername/voxctrl
cd voxctrl
go build -o voxctrl ./cmd/voxctrl
sudo cp voxctrl /usr/local/bin/voxctrl
```

**Step 5 — Install as a systemd user service**

```bash
mkdir -p ~/.config/systemd/user
cp deploy/voxctrl.service ~/.config/systemd/user/
systemctl --user enable voxctrl
systemctl --user start voxctrl
```

The daemon starts automatically on every login. View logs with:

```bash
journalctl --user -u voxctrl -f
```

---

## Distro Compatibility

| Distro | X11 | Wayland | Notes |
|---|---|---|---|
| Ubuntu 20.04 | ✅ | ✅ | |
| Ubuntu 22.04+ | ✅ | ✅ | |
| Debian 11 / 12 | ✅ | ✅ | |
| Fedora 38+ | ✅ | ✅ | |
| Arch Linux | ✅ | ✅ | |
| Manjaro | ✅ | ✅ | |
| Linux Mint | ✅ | ✅ | |
| Pop!\_OS | ✅ | ✅ | |
| openSUSE | ✅ | ✅ | |
| Raspberry Pi OS | ✅ | ✅ | ARM build supported |

VoxCtrl uses evdev (`/dev/input`) for hotkey detection — it has no dependency on X11 or any specific display server.

---

## Configuration

All commands live in `config/commands.yaml`:

```yaml
- name: open vscode
  aliases: ["open vs code", "launch vscode", "start vscode"]
  exec: "code ."

- name: git status
  aliases: ["show git status", "what's the status"]
  exec: "notify-send 'Git Status' \"$(git status --short 2>&1)\""
```

Add a new entry, save the file — the daemon hot-reloads it within 2 seconds. No restart needed.

---

## Project Structure

```
voxctrl/
├── cmd/voxctrl/main.go          # Daemon entry point — pipeline wiring
├── internal/
│   ├── hotkey/                  # evdev-based global hotkey listener
│   ├── audio/                   # Microphone capture via arecord
│   ├── stt/                     # Whisper transcription
│   ├── intent/                  # Fuzzy command matching + YAML registry
│   ├── executor/                # Action interface + shell command runner
│   ├── session/                 # SQLite session logging
│   └── notify/                  # Desktop notification wrapper
├── config/commands.yaml         # Command definitions
└── deploy/voxctrl.service       # systemd unit file
```

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| Speech-to-text | whisper.cpp (local, CPU-only) |
| Hotkey detection | evdev (kernel input, X11 + Wayland) |
| Audio capture | arecord (ALSA) |
| Command matching | Fuzzy search (Levenshtein distance) |
| Session storage | SQLite |
| Notifications | libnotify |
| Service management | systemd user service |

---

*VoxCtrl v1.0 — built to be extended.*