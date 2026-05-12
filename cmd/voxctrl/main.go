package main

import (
	"log"
	"os"
	"os/signal"
	"runtime"
	"strconv"
	"syscall"

	"github.com/dawgdevv/voxctrl/internal/audio"
	"github.com/dawgdevv/voxctrl/internal/executor"
	"github.com/dawgdevv/voxctrl/internal/hotkey"
	"github.com/dawgdevv/voxctrl/internal/intent"
	"github.com/dawgdevv/voxctrl/internal/session"
	"github.com/dawgdevv/voxctrl/internal/stt"
	"github.com/dawgdevv/voxctrl/internal/tray"
)

func main() {
	log.Println("[VoxCtrl] Starting daemon...")

	// ── Config from environment ──────────────────────────────────────────────
	inputDevice := os.Getenv("VOXCTRL_DEVICE")
	alsaDevice := os.Getenv("VOXCTRL_ALSA_DEVICE")

	// ── System Tray (replaces notify-send) ───────────────────────────────────
	trayMgr := tray.New()
	go trayMgr.Run()

	// ── Initialize subsystems ────────────────────────────────────────────────
	store, err := session.NewStore()
	if err != nil {
		log.Fatalf("[VoxCtrl] Failed to init session store: %v", err)
	}
	defer store.Close()

	registry, err := intent.NewRegistry("config/commands.yaml")
	if err != nil {
		log.Fatalf("[VoxCtrl] Failed to load command registry: %v", err)
	}

	parser := intent.NewParser(registry)
	runner := executor.NewRunner(trayMgr)

	// ── Transcriber: prefer resident whisper-server, fall back to CLI ─────────
	modelPath := envWithDefault("VOXCTRL_MODEL", "/usr/local/share/whisper/ggml-base.en.bin")
	threads := envIntWithDefault("VOXCTRL_WHISPER_THREADS", runtime.NumCPU())
	beamSize := envIntWithDefault("VOXCTRL_WHISPER_BEAM", 1)
	gpuLayers := envIntWithDefault("VOXCTRL_GPU_LAYERS", 0)

	var transcriber stt.Transcriber

	serverWhisper := stt.NewServerWhisper(modelPath, "", threads, beamSize, gpuLayers)
	if err := serverWhisper.Start(); err != nil {
		log.Printf("[stt] Server mode unavailable (%v), falling back to whisper-cli", err)
		trayMgr.Logf("STT: falling back to whisper-cli (server unavailable)")
		transcriber = stt.NewWhisper(modelPath, threads, beamSize)
	} else {
		log.Printf("[stt] Whisper server ready on :8080 (threads=%d beam=%d gpu-layers=%d)", threads, beamSize, gpuLayers)
		trayMgr.Logf("STT: whisper-server ready on :8080")
		transcriber = serverWhisper
		defer serverWhisper.Stop()
	}

	recorder := audio.NewRecorder(alsaDevice)

	// ── Channels — the pipeline backbone ─────────────────────────────────────
	hotkeyPress := make(chan bool, 1)
	hotkeyRelease := make(chan bool, 1)
	audioReady := make(chan string, 3)
	transcript := make(chan string, 3)
	actionReady := make(chan executor.Action, 10)

	// ── Hotkey listener ───────────────────────────────────────────────────────
	hk, err := hotkey.NewListener(hotkeyPress, hotkeyRelease, inputDevice)
	if err != nil {
		log.Fatalf("[VoxCtrl] Hotkey init failed: %v\n"+
			"Hint: make sure you are in the 'input' group:\n"+
			"  sudo usermod -aG input $USER  (then log out and back in)\n"+
			"Or set VOXCTRL_DEVICE=/dev/input/eventX manually.", err)
	}
	go hk.Listen()

	// ── Stage 1: Audio capture ────────────────────────────────────────────────
	go func() {
		for {
			<-hotkeyPress
			log.Println("[audio] Hotkey pressed — starting capture")
			trayMgr.Active()

			wavPath, err := recorder.Record(hotkeyRelease)
			if err != nil {
				log.Printf("[audio] Capture error: %v", err)
				trayMgr.Error("Audio capture failed: " + err.Error())
				continue
			}

			select {
			case audioReady <- wavPath:
			default:
				log.Println("[audio] STT stage busy — dropping audio")
				trayMgr.Error("Still processing previous command")
			}
		}
	}()

	// ── Stage 2: Speech-to-text ───────────────────────────────────────────────
	go func() {
		for wavPath := range audioReady {
			log.Println("[stt] Transcribing...")

			text, err := transcriber.Transcribe(wavPath)
			if err != nil {
				log.Printf("[stt] Error: %v", err)
				trayMgr.Error("Transcription failed")
				continue
			}

			if text == "" {
				log.Println("[stt] Empty transcript — nothing spoken?")
				trayMgr.Info("Didn't catch that — try again")
				continue
			}

			log.Printf("[stt] Transcript: %q", text)

			select {
			case transcript <- text:
			default:
				log.Println("[stt] Intent stage busy — dropping transcript")
			}
		}
	}()

	// ── Stage 3: Intent resolution ────────────────────────────────────────────
	go func() {
		for text := range transcript {
			action, confidence, err := parser.Resolve(text)
			if err != nil {
				log.Printf("[intent] No match for %q", text)
				trayMgr.Error("Not recognised: \"" + text + "\"")
				continue
			}

			log.Printf("[intent] Matched %q → %q (confidence=%.2f)", text, action.Name(), confidence)

			// Blocking send — never drop a matched command.
			actionReady <- action
		}
	}()

	// ── Stage 4: Execution + session logging ──────────────────────────────────
	go func() {
		for action := range actionReady {
			entry := store.NewEntry(action.Name())
			err := runner.Run(action)
			entry.Finish(err)

			if logErr := store.Save(entry); logErr != nil {
				log.Printf("[session] Failed to log entry: %v", logErr)
			}
		}
	}()

	// ── Ready ─────────────────────────────────────────────────────────────────
	trayMgr.Logf("VoxCtrl ready — hold Ctrl+Alt to speak")
	log.Println("[VoxCtrl] Daemon running. Ctrl+C to stop.")

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)

	select {
	case <-sig:
	case <-trayMgr.WaitForQuit():
	}

	log.Println("[VoxCtrl] Shutting down.")
}

func envWithDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func envIntWithDefault(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
