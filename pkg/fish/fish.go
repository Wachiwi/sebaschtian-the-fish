//go:build linux

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

	"github.com/hajimehoshi/go-mp3"
	"github.com/hajimehoshi/oto/v2"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/piper"
	"github.com/wachiwi/sebaschtian-the-fish/pkg/playlist"
	"github.com/warthog618/go-gpiocdev"
	"github.com/warthog618/go-gpiocdev/device/rpi"
	"github.com/youpy/go-wav"
)

// Motor represents a single DC motor controlled by an H-Bridge.
type Motor struct {
	enable *gpiocdev.Line
	in1    *gpiocdev.Line
	in2    *gpiocdev.Line
}

// Forward turns the motor in the forward direction.
func (m *Motor) Forward() error {
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

func (fish *Fish) PlaySoundFile(filename string) {
	soundDir := "/sound-data"
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
	myFish.PlayAudioWithAnimation(pcmData, 22050, 1)
	log.Printf("finished saying '%s'.", text)
}

func (fish *Fish) PlayAudioWithAnimation(pcmData []byte, sampleRate, channelCount int) {
	format := oto.FormatSignedInt16LE
	bitDepthInBytes := 2 // 16-bit audio

	otoCtx, ready, err := oto.NewContext(sampleRate, channelCount, format)
	if err != nil {
		log.Printf("failed to create new oto context: %v", err)
		return
	}
	<-ready

	player := otoCtx.NewPlayer(bytes.NewReader(pcmData))
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
