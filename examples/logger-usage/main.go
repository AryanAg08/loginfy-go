package main

import (
	"fmt"
	"time"

	"github.com/AryanAg08/loginfy-go/pkg/logger"
)

func main() {
	// Example 1: Basic logging with default logger
	fmt.Println("\n=== Example 1: Basic Logging ===")
	logger.Infof("Application started at %s", time.Now().Format(time.RFC3339))
	logger.InfoMsg("Processing request", map[string]interface{}{
		"user_id": "u123",
		"action":  "login",
	})

	// Example 2: Service-specific logger
	fmt.Println("\n=== Example 2: Service Logger ===")
	apiLogger := logger.NewServiceLogger("api-gateway", logger.Config{
		Level:    logger.DEBUG,
		UseColor: true,
	})

	apiLogger.Debug("Incoming request")
	apiLogger.Info("Request routed to auth service")
	apiLogger.Warn("High latency detected", map[string]interface{}{
		"duration_ms": 1500,
		"threshold":   1000,
	})

	// Example 3: Session logging with temp files
	fmt.Println("\n=== Example 3: Session Logging ===")
	sessionLogger := logger.NewServiceLogger("order-service")

	// Start a session - creates a temp file
	sess, err := sessionLogger.Logger().StartSession("order-12345")
	if err != nil {
		panic(err)
	}
	defer sess.End()

	fmt.Printf("Session log file: %s\n\n", sess.FilePath)

	sess.Info("Order processing started", map[string]interface{}{
		"order_id": "order-12345",
		"user_id":  "u456",
	})

	// Simulate order processing
	time.Sleep(100 * time.Millisecond)
	sess.Debug("Validating inventory")

	time.Sleep(100 * time.Millisecond)
	sess.Debug("Processing payment")

	time.Sleep(100 * time.Millisecond)
	sess.Info("Order completed successfully", map[string]interface{}{
		"order_id":  "order-12345",
		"total_usd": 99.99,
	})

	// sess.End() is called by defer

	// Example 4: Different log levels
	fmt.Println("\n=== Example 4: Log Levels ===")
	productLogger := logger.NewServiceLogger("product-service", logger.Config{
		Level: logger.WARN, // Only WARN, ERROR, FATAL will be logged
	})

	productLogger.Debug("This won't appear (below threshold)")
	productLogger.Info("This won't appear either (below threshold)")
	productLogger.Warn("Low stock alert!")
	productLogger.Error("Failed to fetch product", map[string]interface{}{
		"product_id": "p789",
		"error":      "connection timeout",
	})

	// Example 5: Multiple active sessions
	fmt.Println("\n=== Example 5: Multiple Sessions ===")
	batchLogger := logger.NewServiceLogger("batch-processor")

	fmt.Printf("Active sessions before: %d\n", batchLogger.Logger().ActiveSessions())

	// Process 3 items in parallel (simulated)
	for i := 1; i <= 3; i++ {
		sessionID := fmt.Sprintf("batch-item-%d", i)
		sess, _ := batchLogger.Logger().StartSession(sessionID)
		defer sess.End()

		sess.Info("Processing started")
		time.Sleep(50 * time.Millisecond)
		sess.Info("Processing completed")
	}

	fmt.Printf("Active sessions during: %d\n", batchLogger.Logger().ActiveSessions())

	// Example 6: JSON output mode (for production log aggregation)
	fmt.Println("\n=== Example 6: JSON Output ===")
	prodLogger := logger.NewServiceLogger("payment-service", logger.Config{
		Level:      logger.INFO,
		JSONOutput: true,
		UseColor:   false,
	})

	prodLogger.Info("Payment processed", map[string]interface{}{
		"transaction_id": "txn-999",
		"amount":         250.00,
		"currency":       "USD",
	})

	fmt.Println("\n=== Examples Complete ===")
}
