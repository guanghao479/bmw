package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

func main() {
	// Force AWS profile if not set
	if os.Getenv("AWS_PROFILE") == "" {
		os.Setenv("AWS_PROFILE", "bmw")
	}

	fmt.Printf("Using AWS Profile: %s\n", os.Getenv("AWS_PROFILE"))

	// Load AWS configuration with explicit profile
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("us-west-2"),
		config.WithSharedConfigProfile(os.Getenv("AWS_PROFILE")),
	)
	if err != nil {
		log.Fatalf("Failed to load AWS config: %v", err)
	}

	// Create DynamoDB client
	dynamoClient := dynamodb.NewFromConfig(cfg)

	// Test table names
	tableNames := []string{
		"seattle-family-activities",
		"seattle-source-management", 
		"seattle-scraping-operations",
	}

	ctx := context.Background()

	fmt.Println("\n=== Testing DynamoDB Connectivity ===")
	for _, tableName := range tableNames {
		fmt.Printf("Testing table: %s\n", tableName)
		
		// Simple scan with limit 1 to test connectivity
		result, err := dynamoClient.Scan(ctx, &dynamodb.ScanInput{
			TableName: aws.String(tableName),
			Limit:     aws.Int32(1),
		})
		
		if err != nil {
			log.Printf("❌ Failed to scan table %s: %v\n", tableName, err)
		} else {
			fmt.Printf("✅ Successfully connected to table %s (found %d items)\n", tableName, len(result.Items))
		}
	}

	fmt.Println("\n=== DynamoDB Connectivity Test Complete ===")
}