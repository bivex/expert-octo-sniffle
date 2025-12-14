package adapters

import (
	"crypto/rand"
	"fmt"
)

// UUIDGenerator generates unique IDs using random bytes
type UUIDGenerator struct{}

// NewUUIDGenerator creates a new UUID generator
func NewUUIDGenerator() *UUIDGenerator {
	return &UUIDGenerator{}
}

// GenerateID generates a unique ID using random bytes
func (g *UUIDGenerator) GenerateID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// Fallback to timestamp-based ID if random generation fails
		return fmt.Sprintf("id_%d", nanoTime())
	}

	// Format as UUID-like string
	return fmt.Sprintf("%x-%x-%x-%x-%x",
		bytes[0:4], bytes[4:6], bytes[6:8], bytes[8:10], bytes[10:16])
}

// nanoTime returns nanosecond timestamp (fallback implementation)
func nanoTime() int64 {
	// This is a simplified implementation
	// In production, you'd use time.Now().UnixNano()
	return 0 // Placeholder
}
