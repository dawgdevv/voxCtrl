# voxCtrl

A local, always-on voice command daemon for Linux. Press a hotkey, speak a command, watch it execute. No cloud. No internet. No data leaves your machine.

Works on any Linux distribution — Ubuntu, Fedora, Arch, Debian, Manjaro, and more. Compatible with both X11 and Wayland.

---

## What It Does

VoxCtrl sits silently in the background as a system daemon. When you hold `Ctrl+Alt` and speak, it records your voice, transcribes it locally using Whisper, matches it to a command, and runs it — all in under 1.5 seconds.

Every interaction is logged to a local SQLite database with full context: what you said, what ran, which window was active, and how long it took.

---

## How It Works

```
Hold Ctrl+Alt  →  speak  →  release  →  command executes
```

Internally, five subsystems chain together via Go channels:

```
Hotkey Listener → Audio Capture → Whisper Server → Intent Parser → Command Executor
                                    (HTTP, resident)                     ↓
                                                               Session Logger (SQLite)
```

Each subsystem is a separate Go package with a clean interface. The hotkey listener reads directly from `/dev/input` via evdev — no dependency on X11 or any display server. This is what makes VoxCtrl work identically across all Linux environments.

> **Note on STT latency:** VoxCtrl starts a `whisper-server` on startup and keeps the model resident in RAM. Audio is sent over HTTP for transcription — no per-command model reload. If the server binary is missing, it automatically falls back to `whisper-cli` (slower but works).

---

## What You Can Say

**System**
- `"lock screen"` — locks the session
- `"volume up"` / `"volume down"` — adjusts system volume
- `"mute"` — toggles mute
- `"take screenshot"` — captures the screen
- `"screenshot area"` — captures a selected region

**Apps**
- `"open terminal"` — opens a new terminal window
- `"open browser"` — launches Firefox
- `"open files"` — opens the file manager
- `"open calendar"` — opens the calendar
- `"open calculator"` — opens the calculator

**Spotify / Media**
- `"open spotify"` — launches Spotify
- `"play music"` / `"pause music"` — controls playback
- `"next song"` / `"previous song"` — skips tracks
- `"now playing"` — shows the current artist and title

**System Info**
- `"show ip"` — displays your local IP
- `"show memory"` — shows RAM usage
- `"show disk usage"` — shows disk space
- `"show uptime"` — shows how long the system has been running

Commands are fuzzy-matched, so `"hey open spotify"` and `"launch browser"` both work. All commands are defined in `config/commands.yaml` — edit the file and changes take effect immediately without restarting the daemon.

---

## What Gets Logged

Every command creates a record in `~/.local/share/voxctrl/sessions.db`:

| Field | Example |
|---|---|
| Transcript | `"open spotify"` |
| Matched intent | `open spotify` |
| Match confidence | `0.94` |
| Result | `success` |
| Active window | `Terminal` |
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
| GTK3 + Ayatana AppIndicator Glib | System tray icon (ships with GNOME, KDE, XFCE) |
| User in `input` group | Read from `/dev/input` for hotkey detection |

**Tools**

| Tool | Install |
|---|---|
| Go 1.21+ | See below |
| whisper-cli / whisper-server | Build from [whisper.cpp](https://github.com/ggml-org/whisper.cpp) |
| arecord | `alsa-utils` package |
| ffmpeg | `ffmpeg` package |

**Build dependency**

| Package | Install |
|---|---|
| `libayatana-appindicator-glib-dev` | `sudo apt install libayatana-appindicator-glib-dev` |

---

## Installation

**Step 1 — Install system dependencies**

```bash
# Debian / Ubuntu
sudo apt install -y golang-go alsa-utils libayatana-appindicator-glib-dev ffmpeg build-essential cmake git

# Fedora / RHEL
sudo dnf install -y golang alsa-utils libayatana-appindicator-glib-devel ffmpeg cmake git gcc gcc-c++

# Arch / Manjaro
sudo pacman -S go alsa-utils libayatana-appindicator-glib ffmpeg cmake git base-devel

# openSUSE
sudo zypper install go alsa-utils libayatana-appindicator-glib-devel ffmpeg cmake git gcc
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
sudo cp ./build/bin/whisper-server /usr/local/bin/whisper-server
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
- name: open spotify
  aliases: ["launch spotify", "play music", "open music"]
  exec: "spotify &"

- name: take screenshot
  aliases: ["screenshot", "capture screen"]
  exec: "gnome-screenshot &"
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
│   └── tray/                    # System tray icon + log viewer
├── config/commands.yaml         # Command definitions
└── deploy/voxctrl.service       # systemd unit file
```

---

## Stack

| Layer | Technology |
|---|---|
| Language | Go |
| Speech-to-text | whisper.cpp server (HTTP, model stays resident) |
| Hotkey detection | evdev (kernel input, X11 + Wayland) |
| Audio capture | arecord (ALSA) |
| Command matching | Fuzzy search (Levenshtein distance) |
| Session storage | SQLite |
| System tray | GTK / Ayatana AppIndicator (native Go) |
| Service management | systemd user service |

---

*VoxCtrl v1.0 — built to be extended.*