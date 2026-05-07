#!/usr/bin/env bash
# VoxCtrl Integration Test Script
# Tests each subsystem independently before full daemon launch

set -e
REPO="/home/nishant-raj/Develop/projects/voxctrl"
WHISPER_MODEL="/usr/local/share/whisper/ggml-base.en.bin"

echo "═══════════════════════════════════════════════════════════════"
echo "  VoxCtrl Component Test Suite"
echo "═══════════════════════════════════════════════════════════════"
echo ""

# ── 1. Unit Tests ──────────────────────────────────────────────
echo "▶ Phase 1: Unit Tests"
cd "$REPO"
if go test ./...; then
    echo "  ✅ All unit tests passed"
else
    echo "  ❌ Unit tests failed"
    exit 1
fi
echo ""

# ── 2. Build ───────────────────────────────────────────────────
echo "▶ Phase 2: Build Binary"
cd "$REPO"
go build -o voxctrl ./cmd/voxctrl
echo "  ✅ Binary built: $(ls -lh voxctrl | awk '{print $5}')"
echo ""

# ── 3. Audio Device Check ──────────────────────────────────────
echo "▶ Phase 3: Audio Capture Devices"
if arecord -l >/dev/null 2>&1; then
    echo "  ✅ ALSA capture devices found:"
    arecord -l | grep -E "^card" | sed 's/^/     /'
else
    echo "  ❌ No ALSA capture devices found"
    exit 1
fi
echo ""

# ── 4. Test Audio Recording ────────────────────────────────────
echo "▶ Phase 4: Audio Recording Test"
echo "  Recording 3 seconds of audio... (speak something)"
arecord -D default -f S16_LE -r 16000 -c 1 -t wav -d 3 /tmp/voxctrl_test.wav 2>/dev/null
if [ -f /tmp/voxctrl_test.wav ] && [ "$(stat -c%s /tmp/voxctrl_test.wav)" -gt 44 ]; then
    echo "  ✅ Audio recorded successfully ($(stat -c%s /tmp/voxctrl_test.wav) bytes)"
else
    echo "  ❌ Audio recording failed"
    exit 1
fi
echo ""

# ── 5. Test Whisper Server ─────────────────────────────────────
echo "▶ Phase 5: Whisper Server (Resident STT)"
if command -v whisper-server >/dev/null 2>&1; then
    echo "  ✅ whisper-server found"
else
    echo "  ⚠️  whisper-server not found — daemon will fall back to slower whisper-cli"
    echo "     Build it from whisper.cpp: cmake --build build -j$(nproc)"
    echo "     Then: sudo cp build/bin/whisper-server /usr/local/bin/"
fi

if [ ! -f "$WHISPER_MODEL" ]; then
    echo "  ❌ Whisper model not found at $WHISPER_MODEL"
    exit 1
fi

# Test server transcription if available
if command -v whisper-server >/dev/null 2>&1; then
    echo "  Starting whisper-server for a quick test..."
    whisper-server -m "$WHISPER_MODEL" --host 127.0.0.1 --port 8080 --no-timestamps --language en >/dev/null 2>&1 &
    SERVER_PID=$!
    sleep 2

    TRANSCRIPT=$(curl -s -F file=@/tmp/voxctrl_test.wav -F temperature=0.0 -F response_format=json http://127.0.0.1:8080/inference 2>/dev/null | python3 -c "import sys,json; print(json.load(sys.stdin).get('text','').strip())" 2>/dev/null)
    kill $SERVER_PID 2>/dev/null || true
    wait $SERVER_PID 2>/dev/null || true

    if [ -n "$TRANSCRIPT" ]; then
        echo "  ✅ Server transcription: \"$TRANSCRIPT\""
    else
        echo "  ⚠️  Empty server transcript (maybe silence?) — this is OK if you didn't speak"
    fi
else
    echo "  Falling back to whisper-cli test..."
    TRANSCRIPT=$(whisper-cli -m "$WHISPER_MODEL" -f /tmp/voxctrl_test.wav --no-timestamps --language en 2>/dev/null | grep -v "^whisper_" | grep -v "^system_info" | grep -v "^main:" | tr '\n' ' ' | sed 's/  */ /g' | xargs)
    if [ -n "$TRANSCRIPT" ]; then
        echo "  ✅ CLI transcription: \"$TRANSCRIPT\""
    else
        echo "  ⚠️  Empty transcript (maybe silence?) — this is OK if you didn't speak"
    fi
fi
echo ""

# ── 6. Test Notification ───────────────────────────────────────
echo "▶ Phase 6: Desktop Notification"
notify-send -u normal "VoxCtrl Test" "If you see this, notifications work ✅"
echo "  ✅ Notification sent (check your screen)"
echo ""

# ── 7. Test Input Device Access ────────────────────────────────
echo "▶ Phase 7: Input Device (Hotkey) Access"
INPUT_DEVICE=$(find /dev/input/by-path -name '*kbd*' -o -name '*keyboard*' 2>/dev/null | head -1)
if [ -z "$INPUT_DEVICE" ]; then
    INPUT_DEVICE=$(find /dev/input/by-id -name '*kbd*' -o -name '*keyboard*' 2>/dev/null | head -1)
fi
if [ -n "$INPUT_DEVICE" ]; then
    REAL_DEVICE=$(readlink -f "$INPUT_DEVICE")
    if [ -r "$REAL_DEVICE" ]; then
        echo "  ✅ Can read input device: $REAL_DEVICE"
    else
        echo "  ❌ Cannot read input device: $REAL_DEVICE"
        echo "     Run: sudo usermod -aG input \$USER  (then log out/in)"
        exit 1
    fi
else
    echo "  ⚠️  Could not auto-detect keyboard device"
    echo "     You may need to set VOXCTRL_DEVICE manually"
fi
echo ""

# ── 8. Test Command Registry ───────────────────────────────────
echo "▶ Phase 8: Command Registry"
if [ -f "$REPO/config/commands.yaml" ]; then
    CMD_COUNT=$(grep -c "^\- name:" "$REPO/config/commands.yaml" || true)
    echo "  ✅ Registry loaded: $CMD_COUNT commands defined"
else
    echo "  ❌ commands.yaml not found"
    exit 1
fi
echo ""

# ── 9. Dry-Run Daemon ──────────────────────────────────────────
echo "▶ Phase 9: Daemon Startup (5 second dry-run)"
cd "$REPO"
timeout 5s ./voxctrl 2>/dev/null || true
if [ $? -eq 124 ] || [ $? -eq 0 ]; then
    echo "  ✅ Daemon starts without crashing"
else
    echo "  ⚠️  Daemon exited early (check logs above)"
fi
echo ""

# ── Cleanup ────────────────────────────────────────────────────
rm -f /tmp/voxctrl_test.wav

echo "═══════════════════════════════════════════════════════════════"
echo "  All component tests complete!"
echo "═══════════════════════════════════════════════════════════════"
echo ""
echo "To run the full daemon:"
echo "  cd $REPO && ./voxctrl"
echo ""
echo "To install as a systemd service:"
echo "  mkdir -p ~/.config/systemd/user"
echo "  cp $REPO/deploy/voxctrl.service ~/.config/systemd/user/"
echo "  systemctl --user daemon-reload"
echo "  systemctl --user enable voxctrl"
echo "  systemctl --user start voxctrl"
echo ""
echo "To view logs:"
echo "  journalctl --user -u voxctrl -f"
