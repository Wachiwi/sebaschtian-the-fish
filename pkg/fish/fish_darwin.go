//go:build darwin

package fish

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
	"github.com/youpy/go-wav"
)

// Motor represents a mock motor for macOS (no GPIO).
type Motor struct{}

// Forward is a no-op on macOS.
func (m *Motor) Forward() error {
	log.Println("[MOCK] Motor forward")
	return nil
}

// Reverse is a no-op on macOS.
func (m *Motor) Reverse() error {
	log.Println("[MOCK] Motor reverse")
	return nil
}

// Stop is a no-op on macOS.
func (m *Motor) Stop() error {
	log.Println("[MOCK] Motor stop")
	return nil
}

// Fish represents the fish with its controllable parts (mock version for macOS).
type Fish struct {
	mu        sync.Mutex
	HeadMotor *Motor
	BodyMotor *Motor
	otoCtx    *oto.Context
}

// NewFish initializes a mock Fish object for macOS.
func NewFish(chipName string) (*Fish, error) {
	log.Println("[MOCK] Initializing fish without GPIO (Darwin/macOS)")

	// Initialize oto context once for the lifetime of the Fish
	op := &oto.NewContextOptions{
		SampleRate:   44100, // Default sample rate
		ChannelCount: 2,     // Default stereo
		Format:       oto.FormatSignedInt16LE,
	}

	otoCtx, ready, err := oto.NewContext(op)
	if err != nil {
		return nil, fmt.Errorf("failed to create oto context: %w", err)
	}
	<-ready

	fish := &Fish{
		HeadMotor: &Motor{},
		BodyMotor: &Motor{},
		otoCtx:    otoCtx,
	}

	return fish, nil
}

func (f *Fish) Lock() {
	f.mu.Lock()
}

func (f *Fish) Unlock() {
	f.mu.Unlock()
}

// Close releases all resources (no-op on macOS).
func (f *Fish) Close() {
	log.Println("[MOCK] Closing fish")
}

func (fish *Fish) PlaySoundFile(filename string) {
	soundDir := "./sound-data"
	filePath := filepath.Join(soundDir, filename)

	log.Printf("playing '%s'...", filename)

	// Add to played list
	item := playlist.PlayedItem{
		Name:      filename,
		Type:      "song",
		Timestamp: time.Now(),
	}
	if err := playlist.AddPlayedItem(item, 1*time.Hour); err != nil {
		log.Printf("Error adding played item: %v", err)
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		log.Printf("failed to read sound file '%s': %v", filePath, err)
		return
	}

	var pcmData []byte
	var sampleRate int
	var channelCount int

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		wavReader := wav.NewReader(bytes.NewReader(fileData))
		format, err := wavReader.Format()
		if err != nil {
			log.Printf("failed to get wav format from '%s': %v", filename, err)
			return
		}
		wavReader = wav.NewReader(bytes.NewReader(fileData))
		pcmData, err = io.ReadAll(wavReader)
		if err != nil {
			log.Printf("failed to decode wav data from '%s': %v", filename, err)
			return
		}
		sampleRate = int(format.SampleRate)
		channelCount = int(format.NumChannels)

	case ".mp3":
		decoder, err := mp3.NewDecoder(bytes.NewReader(fileData))
		if err != nil {
			log.Printf("failed to create mp3 decoder for '%s': %v", filename, err)
			return
		}
		pcmData, err = io.ReadAll(decoder)
		if err != nil {
			log.Printf("failed to decode mp3 data from '%s': %v", filename, err)
			return
		}
		sampleRate = decoder.SampleRate()
		channelCount = 2
	}

	if len(pcmData) > 0 {
		// Convert audio to match oto context (44100Hz stereo)
		if sampleRate != 44100 || channelCount != 2 {
			pcmData = convertAudio(pcmData, sampleRate, channelCount, 44100, 2)
			sampleRate = 44100
			channelCount = 2
		}
		fish.PlayAudioWithAnimation(pcmData, sampleRate, channelCount)
		log.Printf("finished playing '%s'.", filename)
	}
}

func (myFish *Fish) Say(piperClient *piper.PiperClient, text string) {
	if text == "" {
		log.Println("nothing to say.")
		return
	}
	log.Printf("saying '%s'...", text)
	wavData, err := piperClient.Synthesize(text)
	if err != nil {
		log.Printf("failed to synthesize text: %v", err)
		return
	}

	wavReader := wav.NewReader(bytes.NewReader(wavData))
	pcmData, err := io.ReadAll(wavReader)
	if err != nil {
		log.Printf("failed to read pcm data: %v", err)
		return
	}

	// Convert mono to stereo and resample from 22050Hz to 44100Hz
	pcmData = convertAudio(pcmData, 22050, 1, 44100, 2)
	myFish.PlayAudioWithAnimation(pcmData, 44100, 2)
	log.Printf("finished saying '%s'.", text)
}

func (fish *Fish) PlayAudioWithAnimation(pcmData []byte, sampleRate, channelCount int) {
	bitDepthInBytes := 2 // 16-bit audio

	player := fish.otoCtx.NewPlayer(bytes.NewReader(pcmData))
	defer player.Close()
	player.Play()

	go func() {
		const chunkDuration = 100 * time.Millisecond
		const amplitudeThreshold = 1500
		isMouthOpen := false
		chunkSize := int(float64(sampleRate) * chunkDuration.Seconds() * float64(bitDepthInBytes*channelCount))
		buffer := make([]byte, chunkSize)
		analysisReader := bytes.NewReader(pcmData)

		for {
			n, err := io.ReadFull(analysisReader, buffer)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				log.Printf("Error reading audio for animation: %v", err)
				break
			}

			var sum int64
			var count int
			for i := 0; i < n; i += bitDepthInBytes * channelCount {
				if i+(bitDepthInBytes*channelCount) > n {
					break
				}
				sample := int16(binary.LittleEndian.Uint16(buffer[i : i+2]))
				if sample < 0 {
					sum += int64(-sample)
				} else {
					sum += int64(sample)
				}
				count++
			}

			if count == 0 {
				time.Sleep(chunkDuration)
				continue
			}
			avgAmplitude := sum / int64(count)

			fish.Lock()
			if avgAmplitude > amplitudeThreshold && !isMouthOpen {
				fish.OpenMouth()
				isMouthOpen = true
			} else if avgAmplitude <= amplitudeThreshold && isMouthOpen {
				fish.CloseMouth()
				isMouthOpen = false
			}
			fish.Unlock()
			time.Sleep(chunkDuration)
		}

		fish.Lock()
		if isMouthOpen {
			fish.CloseMouth()
			time.Sleep(1 * time.Second)
			fish.StopMouth()
		}

		fish.StopBody()
		fish.StopMouth()

		fish.Unlock()
	}()

	for player.IsPlaying() {
		time.Sleep(100 * time.Millisecond)
	}
}

// convertAudio converts audio from one format to another (sample rate and channel count)
func convertAudio(pcmData []byte, fromRate, fromChannels, toRate, toChannels int) []byte {
	// Convert bytes to int16 samples
	sampleCount := len(pcmData) / 2
	samples := make([]int16, sampleCount)
	for i := 0; i < sampleCount; i++ {
		samples[i] = int16(binary.LittleEndian.Uint16(pcmData[i*2 : i*2+2]))
	}

	// Step 1: Convert mono to stereo if needed
	var stereoSamples []int16
	if fromChannels == 1 && toChannels == 2 {
		stereoSamples = make([]int16, sampleCount*2)
		for i := 0; i < sampleCount; i++ {
			stereoSamples[i*2] = samples[i]   // Left channel
			stereoSamples[i*2+1] = samples[i] // Right channel (duplicate)
		}
	} else if fromChannels == 2 && toChannels == 2 {
		stereoSamples = samples
	} else {
		// For other conversions, just use the samples as-is
		stereoSamples = samples
	}

	// Step 2: Resample if needed
	var resampledSamples []int16
	if fromRate != toRate {
		ratio := float64(toRate) / float64(fromRate)
		newSampleCount := int(float64(len(stereoSamples)) * ratio)
		resampledSamples = make([]int16, newSampleCount)

		for i := 0; i < newSampleCount; i++ {
			// Simple linear interpolation
			srcPos := float64(i) / ratio
			srcIdx := int(srcPos)

			if srcIdx >= len(stereoSamples)-1 {
				resampledSamples[i] = stereoSamples[len(stereoSamples)-1]
			} else {
				frac := srcPos - float64(srcIdx)
				sample1 := float64(stereoSamples[srcIdx])
				sample2 := float64(stereoSamples[srcIdx+1])
				resampledSamples[i] = int16(sample1 + (sample2-sample1)*frac)
			}
		}
	} else {
		resampledSamples = stereoSamples
	}

	// Convert back to bytes
	result := make([]byte, len(resampledSamples)*2)
	for i, sample := range resampledSamples {
		binary.LittleEndian.PutUint16(result[i*2:i*2+2], uint16(sample))
	}

	return result
}

// OpenMouth moves the head motor to open the mouth (mock on macOS).
func (f *Fish) OpenMouth() error {
	return f.HeadMotor.Forward()
}

// CloseMouth moves the head motor to close the mouth (mock on macOS).
func (f *Fish) CloseMouth() error {
	return f.HeadMotor.Reverse()
}

// StopMouth stops the head motor (mock on macOS).
func (f *Fish) StopMouth() error {
	return f.HeadMotor.Stop()
}

// RaiseBody moves the body motor to raise the body (mock on macOS).
func (f *Fish) RaiseBody() error {
	return f.BodyMotor.Forward()
}

// RaiseTail moves the body motor to raise the tail (mock on macOS).
func (f *Fish) RaiseTail() error {
	return f.BodyMotor.Reverse()
}

// StopBody stops the body motor (mock on macOS).
func (f *Fish) StopBody() error {
	return f.BodyMotor.Stop()
}
