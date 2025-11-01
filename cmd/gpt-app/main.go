package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	porcupine "github.com/Picovoice/porcupine/binding/go/v3"
	"github.com/spf13/cobra"
	"google.golang.org/protobuf/types/known/emptypb"

	api_client "github.com/GoBig87/chat-gpt-raspberry-pi-assistant/pkg/api/client"
	v1 "github.com/GoBig87/chat-gpt-raspberry-pi-assistant/pkg/api/v1"
	gpio_motor "github.com/GoBig87/chat-gpt-raspberry-pi-assistant/pkg/gpio-motor"
	ww "github.com/GoBig87/chat-gpt-raspberry-pi-assistant/pkg/wake-word"
)

var (
	accessKey string
	client    *api_client.ApiClient
)

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}

var rootCmd = &cobra.Command{
	Use:           "app",
	Short:         "Main application to handle speech-to-text, chat-gpt and text-to-speech",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		ctx := cmd.Context()
		run(ctx)
		return nil
	},
}

func run(ctx context.Context) {
	for {
		err := process(ctx)
		if err != nil {
			fmt.Printf("Error processing: %v", err)
		}
	}
}

func detectWakeWord(ctx context.Context) (porcupine.BuiltInKeyword, error) {
	stream, err := client.WW.DetectWakeWord(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("Error creating wake word stream: %v", err)
		return "", err
	}
	for {
		resp, err := stream.Recv()
		if err != nil {
			log.Printf("Error receiving wake word, stop to stream: %v", err)
			return "", err
		}
		if resp.Detected {
			log.Printf("Wake word detected: %v", resp.BuiltInKeyword)
			return ww.StringToBuiltInKeyword(resp.BuiltInKeyword), nil
		}
		time.Sleep(250 * time.Millisecond)
	}
}

func process(ctx context.Context) error {
	gpioMotor, err := gpio_motor.MakeNewGpioMotor(
		32, // motorMouthEna,
		35, // motorMouthIn1,
		37, // motorMouthIn2,

		31, // motorBodyIn3,
		33, // motorBodyIn4,
		29, // motorBodyEnb,

		36, // audioDetect
	)
	if err != nil {
		return err
	}
	log.Printf("Open")
	err = gpioMotor.OpenMouth()
	if err != nil {
		return err
	}

	time.Sleep(1 * time.Second)
	log.Printf("Close")
	err = gpioMotor.CloseMouth()
	if err != nil {
		return err
	}
	time.Sleep(1 * time.Second)
	return nil
}

func processSpeechToText(ctx context.Context) (string, error) {
	log.Println("Processing speech to text")
	// 2. Speech-to-text
	req := &v1.ProcessSpeechRequest{
		// TODO UNHARDCODE
		WakeWord: v1.WakeWord_WAKE_WORD_HEY_GOOGLE,
	}
	clnt, err := client.S2T.ProcessSpeech(ctx, req)
	if err != nil {
		log.Printf("error processing speech: %v", err)
		return "", err
	}
	processed := false
	text := ""
	for {
		if processed {
			break
		}
		resp, err := clnt.Recv()
		if err != nil {
			log.Printf("error receiving speech: %v", err)
			return "", err
		}
		if !resp.Processing {
			text = resp.TranscribedText
			processed = true
		}
	}
	if text == "" {
		log.Printf("text is empty")
		return "", fmt.Errorf("text is empty")
	}
	log.Println("finished processing wake word")
	return text, nil
}

func processChatGptPrompt(ctx context.Context, prompt string) (string, error) {
	log.Printf("Processing Prompt: %v\n", prompt)
	// 3. Chat GPT Response
	req := &v1.ProcessPromptRequest{
		Prompt: prompt,
	}
	resp, err := client.GPT.ProcessPrompt(ctx, req)
	if err != nil {
		log.Printf("error processing prompt: %v", err)
		return "", err
	}
	answer := resp.Response
	if answer == "" {
		log.Printf("response is empty")
		return "", fmt.Errorf("response is empty")
	}
	log.Println("finished processing prompt")
	return answer, nil
}

func processChatGptResponse(ctx context.Context, response string) error {
	log.Printf("Processing Response: %v\n", response)
	// 4. Text-to-speech
	req := &v1.ProcessTextRequest{
		Text: response,
	}
	clnt, err := client.T2S.ProcessText(ctx, req)
	if err != nil {
		log.Printf("error processing response: %v", err)
		return err
	}
	for {
		resp, err := clnt.Recv()
		if err != nil {
			log.Printf("error receiving response: %v", err)
			return err
		}
		if resp.Processed {
			break
		}
		time.Sleep(1 * time.Millisecond)
	}
	log.Println("finished processing response")
	return nil
}

func wagTail(ctx context.Context, done chan struct{}) {
	// TODO move this to a GRPC stream
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	_, err := client.MTR.ResetAll(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("Error lowering head: %v", err)
	}
	// Sleep for half a second
	time.Sleep(500 * time.Millisecond)
	raised := false
	for {
		select {
		case <-done:
			if _, err := client.MTR.ResetAll(ctx, &emptypb.Empty{}); err != nil {
				log.Printf("Error reseting: %v", err)
			}
			return
		case <-ticker.C:
			if !raised {
				// Raise the tail
				if _, err := client.MTR.RaiseTail(ctx, &emptypb.Empty{}); err != nil {
					log.Printf("Error raising tail: %v", err)
				} else {
					raised = true
				}
			} else {
				// Lower the tail
				if _, err := client.MTR.LowerTail(ctx, &emptypb.Empty{}); err != nil {
					log.Printf("Error lowering tail: %v", err)
				} else {
					raised = false
				}
			}
		}
	}
}

func moveMouth(ctx context.Context, done chan struct{}) {
	ticker := time.NewTicker(20 * time.Millisecond)
	defer ticker.Stop()
	if _, err := client.MTR.LowerTail(ctx, &emptypb.Empty{}); err != nil {
		log.Printf("Error lowering tail: %v", err)
	}
	time.Sleep(500 * time.Millisecond)
	_, err := client.MTR.RaiseHead(ctx, &emptypb.Empty{})
	if err != nil {
		log.Printf("Error lowering head: %v", err)
	}
	stream, err := client.MTR.MoveMouthToSpeech(ctx)
	if err != nil {
		log.Printf("Error creating moving mouth to speech stream: %v", err)
		return
	}

	for {
		select {
		case <-done:
			req := &v1.MoveMouthToSpeechRequest{
				Stop: true,
			}
			err = stream.Send(req)
			if err != nil {
				log.Printf("Error sending stop to stream: %v", err)
				return
			}
			_, err := stream.CloseAndRecv()
			if err != nil {
				log.Printf("Error closing stream: %v", err)
				return
			}
			if _, err := client.MTR.ResetAll(ctx, &emptypb.Empty{}); err != nil {
				log.Printf("Error reseting: %v", err)
			}
			return
		case <-ticker.C:
			req := &v1.MoveMouthToSpeechRequest{
				Stop: false,
			}
			err = stream.Send(req)
			if err != nil {
				log.Printf("Error sending stop to stream: %v", err)
				return
			}
		}
	}
}
