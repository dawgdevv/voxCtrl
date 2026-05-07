package stt

// Transcriber turns a WAV file into text.
type Transcriber interface {
	Transcribe(wavPath string) (string, error)
}
