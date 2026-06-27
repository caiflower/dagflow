package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	sdk "github.com/caiflower/dagflow/clients/go"
)

type ImageInput struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
}

type ImageOutput struct {
	ProcessedURL string `json:"processedUrl"`
	Size         int64  `json:"size"`
}

func processImage(ctx context.Context, input ImageInput) (ImageOutput, error) {
	fmt.Printf("Processing image: %s (%dx%d)\n", input.URL, input.Width, input.Height)
	return ImageOutput{
		ProcessedURL: input.URL + "?processed=true",
		Size:         int64(input.Width * input.Height * 4),
	}, nil
}

func echo(ctx context.Context, input string) (ImageInput, error) {
	return ImageInput{
		URL:    "http://" + input,
		Width:  1920,
		Height: 1080,
	}, nil
}

func branchSelect(ctx context.Context, input json.RawMessage) (string, error) {
	fmt.Printf("Branch: selecting 'echo', input=%s\n", string(input))
	return "task_1781927318318", nil
}

func processImage1(ctx context.Context, input ImageInput) (ImageOutput, error) {
	fmt.Printf("Processing image1: %s (%dx%d)\n", input.URL, input.Width, input.Height)
	return ImageOutput{
		ProcessedURL: input.URL + "?processed=v1",
		Size:         int64(input.Width * input.Height * 8),
	}, nil
}

func main() {
	s := sdk.New(sdk.Config{
		NodeID:     "demo-node-1",
		EngineAddr: "localhost:50051",
		ListenAddr: "localhost:50052",
	})

	sdk.Register(s, "processImage", processImage)
	sdk.Register(s, "echo", echo)
	sdk.Register(s, "branch", branchSelect)
	sdk.Register(s, "processImage1", processImage1)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		<-sigCh
		cancel()
	}()

	fmt.Println("SDK demo node starting...")
	if err := s.Start(ctx); err != nil {
		fmt.Fprintf(os.Stderr, "SDK error: %v\n", err)
		os.Exit(1)
	}
	fmt.Println("SDK demo node stopped")
}
