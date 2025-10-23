package uuid

import (
	"crypto/rand"
	"fmt"
	"regexp"
)

// generateUUID manually generates a version 4 UUID (based on random data)
func GenerateUUID() string {
	// Create a 16-byte array for the UUID
	uuid := make([]byte, 16)

	// Fill the first 6 bytes with random data
	_, err := rand.Read(uuid)
	if err != nil {
		panic(fmt.Sprintf("error generating random data: %v", err))
	}

	// Set version 4 (random) UUID, so we set the 7th byte's top 4 bits to 0100
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // 0100 for version 4 UUID

	// Set the variant to 10xx for the RFC4122 variant
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // 10xx for the variant

	// Convert the UUID into a string with the format 8-4-4-4-12
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x", uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:])
}

func ParseUUID(uuidStr string) (string, error) {
	// Regular expression for UUID (8-4-4-4-12 format)
	re := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)

	if !re.MatchString(uuidStr) {
		return "", fmt.Errorf("invalid UUID format: %s", uuidStr)
	}
	return uuidStr, nil
}
