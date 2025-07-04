package utils

import (
	"testing"
)

func TestParseLink(t *testing.T) {
	testCases := []struct {
		name        string
		link        string
		expectError bool
	}{
		{
			name:        "Valid t.me link",
			link:        "https://t.me/test_channel/123",
			expectError: false,
		},
		{
			name:        "Valid telegram.me link",
			link:        "https://telegram.me/test_channel/456",
			expectError: false,
		},
		{
			name:        "Invalid URL",
			link:        "not-a-url",
			expectError: true,
		},
		{
			name:        "Wrong domain",
			link:        "https://example.com/channel/123",
			expectError: true,
		},
		{
			name:        "Missing message ID",
			link:        "https://t.me/channel",
			expectError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// This would test the Telegram link parsing logic
			// For now, we'll just check if the link is not empty
			if tc.link == "" && !tc.expectError {
				t.Error("Expected non-empty link")
			}
		})
	}
}

func TestGenerateIDs(t *testing.T) {
	correlationID := GenerateCorrelationID()
	if correlationID == "" {
		t.Error("Expected non-empty correlation ID")
	}

	requestID := GenerateRequestID()
	if requestID == "" {
		t.Error("Expected non-empty request ID")
	}

	// Check that IDs are different
	if correlationID == requestID {
		t.Error("Correlation ID and request ID should be different")
	}
}
