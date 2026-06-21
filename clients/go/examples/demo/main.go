package main

import (
	"context"
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

func main() {
	s := sdk.New(sdk.Config{
		NodeID:     "demo-node-1",
		EngineAddr: "localhost:50051",
		ListenAddr: "localhost:50052",
	})

	sdk.Register(s, "processImage", processImage)

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
