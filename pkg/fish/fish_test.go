package fish

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
)

func TestFishLifecycle(t *testing.T) {
	// 1. Setup Environment
	// Ensure sound-data exists locally for the test
	if err := os.MkdirAll("sound-data", 0755); err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll("sound-data")

	// Create a dummy valid WAV file (Header + 4 bytes silence)
	// 16-bit Mono 44.1kHz
	wavHeader := []byte{
		0x52, 0x49, 0x46, 0x46, // RIFF
		0x28, 0x00, 0x00, 0x00, // ChunkSize (36 + 4 = 40)
		0x57, 0x41, 0x56, 0x45, // WAVE
		0x66, 0x6D, 0x74, 0x20, // fmt
		0x10, 0x00, 0x00, 0x00, // Subchunk1Size (16)
		0x01, 0x00, // AudioFormat (1 = PCM)
		0x01, 0x00, // NumChannels (1)
		0x44, 0xAC, 0x00, 0x00, // SampleRate (44100)
		0x88, 0x58, 0x01, 0x00, // ByteRate (88200)
		0x02, 0x00, // BlockAlign (2)
		0x10, 0x00, // BitsPerSample (16)
		0x64, 0x61, 0x74, 0x61, // data
		0x04, 0x00, 0x00, 0x00, // Subchunk2Size (4)
		0x00, 0x00, 0x00, 0x00, // Silence
	}

	err := os.WriteFile("sound-data/test.wav", wavHeader, 0644)
	if err != nil {
		t.Fatal(err)
	}

	// 2. Initialize Fish
	// Note: This attempts to initialize audio. If no audio device is present (CI/Headless),
	// NewFish might fail. We skip the test in that case.
	f, err := NewFish("mock-chip")
	if err != nil {
		t.Logf("Skipping fish test: Audio initialization failed (expected in headless env): %v", err)
		return
	}
	defer f.Close()

	ctx := context.Background()

	// 3. Test PlaySoundFile
	t.Run("PlaySoundFile", func(t *testing.T) {
		err := f.PlaySoundFile(ctx, "test.wav")
		if err != nil {
			t.Errorf("PlaySoundFile failed: %v", err)
		}
	})

	// 4. Test Say (with mocked Piper)
	t.Run("Say", func(t *testing.T) {
		// Mock Piper server returning our valid WAV
		ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Write(wavHeader)
		}))
		defer ts.Close()

		pClient := piper.NewPiperClient(ts.URL)
		err := f.Say(ctx, pClient, "Hello Fish")
		if err != nil {
			t.Errorf("Say failed: %v", err)
		}
	})

	// 5. Test Motors (Mock implementation on Darwin)
	t.Run("Motors", func(t *testing.T) {
		if err := f.OpenMouth(); err != nil {
			t.Error(err)
		}
		if err := f.CloseMouth(); err != nil {
			t.Error(err)
		}
		if err := f.RaiseBody(); err != nil {
			t.Error(err)
		}
		if err := f.RaiseTail(); err != nil {
			t.Error(err)
		}
		if err := f.StopBody(); err != nil {
			t.Error(err)
		}
	})
}
