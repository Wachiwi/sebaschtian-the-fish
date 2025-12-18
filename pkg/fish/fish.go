//go:build linux

package fish

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ebitengine/oto/v3"
	"github.com/hajimehoshi/go-mp3"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
	"github.com/youpy/go-wav"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

var (
	motorOpsCounter metric.Int64Counter
)

func init() {
	var err error
	meter := otel.Meter("github.com/wachiwi/sebaschtian-the-fish/pkg/fish")
	motorOpsCounter, err = meter.Int64Counter("fish.motor.operations",
		metric.WithDescription("Total number of motor operations"),
		metric.WithUnit("{ops}"),
	)
	if err != nil {
		slog.Error("Failed to create motor metrics", "error", err)
	}
}

// Motor represents a single DC motor controlled by an H-Bridge.
type Motor struct {
	enable *gpiocdev.Line
	in1    *gpiocdev.Line
	in2    *gpiocdev.Line
}

// Forward turns the motor in the forward direction.
func (m *Motor) Forward() error {
	motorOpsCounter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("direction", "forward")))
	if err := m.in1.SetValue(1); err != nil {
		return err
	}
	if err := m.in2.SetValue(0); err != nil {
		return err
	}
	return m.enable.SetValue(1)
}

// Reverse turns the motor in the reverse direction.
func (m *Motor) Reverse() error {
	motorOpsCounter.Add(context.Background(), 1, metric.WithAttributes(attribute.String("direction", "reverse")))
	if err := m.in1.SetValue(0); err != nil {
		return err
	}
	if err := m.in2.SetValue(1); err != nil {
		return err
	}
	return m.enable.SetValue(1)
}

// Stop halts the motor.
func (m *Motor) Stop() error {
	if err := m.in1.SetValue(0); err != nil {
		return err
	}
	if err := m.in2.SetValue(0); err != nil {
		return err
	}
	return m.enable.SetValue(0)
}

// Fish represents the fish with its controllable parts.
type Fish struct {
	mu        sync.Mutex
	chip      *gpiocdev.Chip
	HeadMotor *Motor
	BodyMotor *Motor
	otoCtx    *oto.Context
}

// NewFish initializes the GPIO pins and returns a new Fish object.
func NewFish(chipName string) (*Fish, error) {
	c, err := gpiocdev.NewChip(chipName)
	if err != nil {
		return nil, fmt.Errorf("failed to open chip: %w", err)
	}

	// Head motor pins
	enableHeadPin, err := c.RequestLine(rpi.GPIO5, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in1Pin, err := c.RequestLine(rpi.GPIO13, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in2Pin, err := c.RequestLine(rpi.GPIO6, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}

	// Body motor pins
	enableBodyPin, err := c.RequestLine(rpi.GPIO12, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in3Pin, err := c.RequestLine(rpi.GPIO26, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}
	in4Pin, err := c.RequestLine(rpi.GPIO19, gpiocdev.AsOutput(0))
	if err != nil {
		return nil, err
	}

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
		chip: c,
		HeadMotor: &Motor{
			enable: enableHeadPin,
			in1:    in1Pin,
			in2:    in2Pin,
		},
		BodyMotor: &Motor{
			enable: enableBodyPin,
			in1:    in3Pin,
			in2:    in4Pin,
		},
		otoCtx: otoCtx,
	}

	return fish, nil
}

func (f *Fish) Lock() {
	f.mu.Lock()
}

func (f *Fish) Unlock() {
	f.mu.Unlock()
}

// Close releases all GPIO resources.
func (f *Fish) Close() {
	f.HeadMotor.Stop()
	f.BodyMotor.Stop()
	f.chip.Close()
}

// PlaySoundFile plays a sound file and animates the fish.
// It returns an error if the file cannot be read, decoded, or played.
func (fish *Fish) PlaySoundFile(ctx context.Context, filename string) error {
	ctx, span := otel.Tracer("fish").Start(ctx, "PlaySoundFile")
	defer span.End()
	span.SetAttributes(attribute.String("filename", filename))

	soundDir := "/sound-data"
	filePath := filepath.Join(soundDir, filename)

	slog.Info("playing", "filename", filename)

	// Add to played list
	item := playlist.PlayedItem{
		Name:      filename,
		Type:      "song",
		Timestamp: time.Now(),
	}
	if err := playlist.AddPlayedItem(item, 1*time.Hour); err != nil {
		slog.Error("Error adding played item", "error", err)
		span.RecordError(err)
		// Non-fatal error, continue
	}

	fileData, err := os.ReadFile(filePath)
	if err != nil {
		err = fmt.Errorf("failed to read sound file '%s': %w", filePath, err)
		span.RecordError(err)
		return err
	}

	var pcmData []byte
	var sampleRate int
	var channelCount int

	switch strings.ToLower(filepath.Ext(filename)) {
	case ".wav":
		wavReader := wav.NewReader(bytes.NewReader(fileData))
		format, err := wavReader.Format()
		if err != nil {
			err = fmt.Errorf("failed to get wav format from '%s': %w", filename, err)
			span.RecordError(err)
			return err
		}
		wavReader = wav.NewReader(bytes.NewReader(fileData))
		pcmData, err = io.ReadAll(wavReader)
		if err != nil {
			err = fmt.Errorf("failed to decode wav data from '%s': %w", filename, err)
			span.RecordError(err)
			return err
		}
		sampleRate = int(format.SampleRate)
		channelCount = int(format.NumChannels)

	case ".mp3":
		decoder, err := mp3.NewDecoder(bytes.NewReader(fileData))
		if err != nil {
			err = fmt.Errorf("failed to create mp3 decoder for '%s': %w", filename, err)
			span.RecordError(err)
			return err
		}
		pcmData, err = io.ReadAll(decoder)
		if err != nil {
			err = fmt.Errorf("failed to decode mp3 data from '%s': %w", filename, err)
			span.RecordError(err)
			return err
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
		if err := fish.PlayAudioWithAnimation(ctx, pcmData, sampleRate, channelCount); err != nil {
			err = fmt.Errorf("failed to play audio: %w", err)
			span.RecordError(err)
			return err
		}
		slog.Info("finished playing", "filename", filename)
	}
	return nil
}

// Say synthesizes text and plays it while animating the fish.
// It returns an error if synthesis or playback fails.
func (myFish *Fish) Say(ctx context.Context, piperClient *piper.PiperClient, text string) error {
	ctx, span := otel.Tracer("fish").Start(ctx, "Say")
	defer span.End()
	span.SetAttributes(attribute.String("text_length", fmt.Sprintf("%d", len(text))))

	if text == "" {
		slog.Info("nothing to say.")
		return nil
	}
	slog.Info("saying", "text", text)
	wavData, err := piperClient.Synthesize(text)
	if err != nil {
		err = fmt.Errorf("failed to synthesize text: %w", err)
		span.RecordError(err)
		return err
	}

	wavReader := wav.NewReader(bytes.NewReader(wavData))
	pcmData, err := io.ReadAll(wavReader)
	if err != nil {
		err = fmt.Errorf("failed to read pcm data: %w", err)
		span.RecordError(err)
		return err
	}

	// Convert mono to stereo and resample from 22050Hz to 44100Hz
	pcmData = convertAudio(pcmData, 22050, 1, 44100, 2)
	if err := myFish.PlayAudioWithAnimation(ctx, pcmData, 44100, 2); err != nil {
		err = fmt.Errorf("failed to play audio: %w", err)
		span.RecordError(err)
		return err
	}
	slog.Info("finished saying", "text", text)
	return nil
}

// PlayAudioWithAnimation plays audio and animates the mouth.
// It implements a timeout to prevent hanging if the audio player gets stuck.
func (fish *Fish) PlayAudioWithAnimation(ctx context.Context, pcmData []byte, sampleRate, channelCount int) error {
	_, span := otel.Tracer("fish").Start(ctx, "PlayAudioWithAnimation")
	defer span.End()

	bitDepthInBytes := 2 // 16-bit audio

	// Calculate approximate duration for timeout
	numSamples := len(pcmData) / (channelCount * bitDepthInBytes)
	duration := time.Duration(float64(numSamples) / float64(sampleRate) * float64(time.Second))

	player := fish.otoCtx.NewPlayer(bytes.NewReader(pcmData))
	defer player.Close()
	player.Play()

	done := make(chan struct{})

	go func() {
		defer close(done)
		const chunkDuration = 100 * time.Millisecond
		const amplitudeThreshold = 1500
		isMouthOpen := false
		chunkSize := int(float64(sampleRate) * chunkDuration.Seconds() * float64(bitDepthInBytes*channelCount))
		buffer := make([]byte, chunkSize)
		analysisReader := bytes.NewReader(pcmData)

		for {
			// Check if we should stop (if player closed or timeout occurred outside)
			// But since we can't easily check 'done' from outside here, we rely on read finishing.
			// However, if we need to abort, we rely on the loop finishing when analysisReader is done.

			n, err := io.ReadFull(analysisReader, buffer)
			if err == io.EOF || err == io.ErrUnexpectedEOF {
				break
			}
			if err != nil {
				slog.Error("Error reading audio for animation", "error", err)
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

	// Safety timeout: Expected duration + 5 seconds margin
	timeout := duration + 5*time.Second
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for player.IsPlaying() {
		select {
		case <-timer.C:
			return fmt.Errorf("playback timed out after %v", timeout)
		case <-ticker.C:
			// continue polling
		}
	}

	// Wait for animation to finish (it shouldn't take long after playback finishes)
	// We use a small timeout here too just in case
	select {
	case <-done:
		return nil
	case <-time.After(2 * time.Second):
		slog.Warn("Animation goroutine took too long to finish")
		return nil
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

// OpenMouth moves the head motor to open the mouth.
func (f *Fish) OpenMouth() error {
	return f.HeadMotor.Forward()
}

// CloseMouth moves the head motor to close the mouth.
func (f *Fish) CloseMouth() error {
	return f.HeadMotor.Reverse()
}

// StopMouth stops the head motor.
func (f *Fish) StopMouth() error {
	return f.HeadMotor.Stop()
}

// RaiseBody moves the body motor to raise the body.
func (f *Fish) RaiseBody() error {
	return f.BodyMotor.Forward()
}

// RaiseTail moves the body motor to raise the tail.
func (f *Fish) RaiseTail() error {
	return f.BodyMotor.Reverse()
}

// StopBody stops the body motor.
func (f *Fish) StopBody() error {
	return f.BodyMotor.Stop()
}
