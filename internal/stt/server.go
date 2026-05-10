package stt

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"time"
)

// ServerWhisper keeps a whisper.cpp server running and talks to it over HTTP.
// This avoids the ~300-800ms model reload cost on every command.
type ServerWhisper struct {
	modelPath string
	serverBin string
	threads   string
	beamSize  string
	gpuLayers string
	serverCmd *exec.Cmd
	client    *http.Client
	baseURL   string
}

// NewServerWhisper creates a ServerWhisper. serverBin can be "" to use "whisper-server" from PATH.
func NewServerWhisper(modelPath, serverBin string, threads, beamSize, gpuLayers int) *ServerWhisper {
	if serverBin == "" {
		serverBin = "whisper-server"
	}
	return &ServerWhisper{
		modelPath: modelPath,
		serverBin: serverBin,
		threads:   fmt.Sprintf("%d", threads),
		beamSize:  fmt.Sprintf("%d", beamSize),
		gpuLayers: fmt.Sprintf("%d", gpuLayers),
		client:    &http.Client{Timeout: 30 * time.Second},
		baseURL:   "http://127.0.0.1:8080",
	}
}

// Start launches whisper-server and blocks until it is accepting requests.
func (s *ServerWhisper) Start() error {
	args := []string{
		"-m", s.modelPath,
		"--host", "127.0.0.1",
		"--port", "8080",
		"-t", s.threads,
		"-bs", s.beamSize,
		"--no-timestamps",
		"--language", "en",
	}
	if s.gpuLayers != "0" {
		args = append(args, "-ngl", s.gpuLayers)
	}
	s.serverCmd = exec.Command(s.serverBin, args...)
	s.serverCmd.Stdout = os.Stdout
	s.serverCmd.Stderr = os.Stderr

	if err := s.serverCmd.Start(); err != nil {
		return fmt.Errorf("start %s: %w", s.serverBin, err)
	}

	// Poll until the server is accepting connections.
	for i := 0; i < 60; i++ {
		time.Sleep(100 * time.Millisecond)
		if s.ready() {
			return nil
		}
	}
	_ = s.Stop()
	return fmt.Errorf("whisper-server did not become ready within 6s")
}

// ready returns true if the server is listening.
func (s *ServerWhisper) ready() bool {
	resp, err := s.client.Get(s.baseURL)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	return resp.StatusCode < 500 // 200 or 404 both mean it's up
}

// Transcribe sends the WAV file to the server and returns the transcript.
func (s *ServerWhisper) Transcribe(wavPath string) (string, error) {
	body, contentType, err := buildMultipartBody(wavPath)
	if err != nil {
		return "", fmt.Errorf("build request: %w", err)
	}

	ctxt, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctxt, "POST", s.baseURL+"/inference", body)
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", contentType)

	resp, err := s.client.Do(req)
	if err != nil {
		return "", fmt.Errorf("http post: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("whisper-server %d: %s", resp.StatusCode, string(respBody))
	}

	return parseServerResponse(respBody)
}

// Stop kills the server subprocess.
func (s *ServerWhisper) Stop() error {
	if s.serverCmd != nil && s.serverCmd.Process != nil {
		return s.serverCmd.Process.Kill()
	}
	return nil
}

func buildMultipartBody(wavPath string) (*bytes.Buffer, string, error) {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	fw, err := w.CreateFormFile("file", wavPath)
	if err != nil {
		return nil, "", err
	}
	f, err := os.Open(wavPath)
	if err != nil {
		return nil, "", err
	}
	defer f.Close()
	if _, err := io.Copy(fw, f); err != nil {
		return nil, "", err
	}

	_ = w.WriteField("temperature", "0.0")
	_ = w.WriteField("response_format", "json")
	_ = w.Close()

	return &b, w.FormDataContentType(), nil
}

func parseServerResponse(data []byte) (string, error) {
	// whisper-server returns JSON with a "text" field.
	var result struct {
		Text string `json:"text"`
	}
	if err := json.Unmarshal(data, &result); err != nil {
		return "", fmt.Errorf("parse json: %w (body: %s)", err, string(data))
	}
	return cleanTranscript(result.Text), nil
}
