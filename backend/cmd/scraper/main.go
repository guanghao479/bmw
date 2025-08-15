package main

import (
    "context"
    "fmt"
    "log"

    "github.com/aws/aws-lambda-go/lambda"
)

// Placeholder handler until we implement the full scraper
func HandleScraping(ctx context.Context) error {
    log.Println("Seattle Family Activities Scraper - Placeholder")
    fmt.Println("Scraper will be implemented in the next steps")
    return nil
}

func main() {
    lambda.Start(HandleScraping)
}