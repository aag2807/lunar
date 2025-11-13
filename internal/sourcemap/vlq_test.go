package sourcemap

import "testing"

func TestEncodeVLQ(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "A"},       // 0
		{1, "C"},       // 1
		{-1, "D"},      // -1
		{2, "E"},       // 2
		{-2, "F"},      // -2
		{15, "e"},      // 15
		{-15, "f"},     // -15
		{16, "gB"},     // 16 (requires continuation)
		{-16, "hB"},    // -16
		{123, "2H"},    // 123
		{-123, "3H"},   // -123
		{1234, "ktC"},  // 1234
		{-1234, "ltC"}, // -1234
	}

	for _, tt := range tests {
		result := EncodeVLQ(tt.input)
		if result != tt.expected {
			t.Errorf("EncodeVLQ(%d): expected '%s', got '%s'", tt.input, tt.expected, result)
		}
	}
}

func TestDecodeVLQ(t *testing.T) {
	tests := []struct {
		input         string
		expectedValue int
		expectedChars int
	}{
		{"A", 0, 1},
		{"C", 1, 1},
		{"D", -1, 1},
		{"E", 2, 1},
		{"F", -2, 1},
		{"e", 15, 1},
		{"f", -15, 1},
		{"gB", 16, 2},
		{"hB", -16, 2},
		{"2H", 123, 2},
		{"3H", -123, 2},
		{"ktC", 1234, 3},
		{"ltC", -1234, 3},
	}

	for _, tt := range tests {
		value, chars := DecodeVLQ(tt.input)
		if value != tt.expectedValue {
			t.Errorf("DecodeVLQ('%s'): expected value %d, got %d", tt.input, tt.expectedValue, value)
		}
		if chars != tt.expectedChars {
			t.Errorf("DecodeVLQ('%s'): expected %d chars read, got %d", tt.input, tt.expectedChars, chars)
		}
	}
}

func TestVLQRoundTrip(t *testing.T) {
	// Test round-trip encoding and decoding
	testValues := []int{
		0, 1, -1, 2, -2, 10, -10, 15, -15, 16, -16,
		100, -100, 123, -123, 1000, -1000, 1234, -1234,
		10000, -10000, 32767, -32768,
	}

	for _, original := range testValues {
		encoded := EncodeVLQ(original)
		decoded, _ := DecodeVLQ(encoded)

		if decoded != original {
			t.Errorf("Round-trip failed for %d: encoded '%s', decoded %d", original, encoded, decoded)
		}
	}
}

func TestDecodeVLQWithTrailingChars(t *testing.T) {
	// Test that DecodeVLQ correctly handles trailing characters
	input := "AACDE"
	value1, chars1 := DecodeVLQ(input)

	if value1 != 0 {
		t.Errorf("First decode: expected 0, got %d", value1)
	}
	if chars1 != 1 {
		t.Errorf("First decode: expected 1 char consumed, got %d", chars1)
	}

	// Decode next value
	value2, chars2 := DecodeVLQ(input[chars1:])
	if value2 != 0 {
		t.Errorf("Second decode: expected 0, got %d", value2)
	}
	if chars2 != 1 {
		t.Errorf("Second decode: expected 1 char consumed, got %d", chars2)
	}
}

func TestBase64CharMapping(t *testing.T) {
	// Verify the base64 character set is correct
	expected := "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	if base64Chars != expected {
		t.Errorf("base64Chars mismatch:\nExpected: %s\nGot:      %s", expected, base64Chars)
	}

	if len(base64Chars) != 64 {
		t.Errorf("base64Chars should have 64 characters, got %d", len(base64Chars))
	}
}

func TestVLQConstants(t *testing.T) {
	if vlqBase != 32 {
		t.Errorf("vlqBase should be 32, got %d", vlqBase)
	}

	if vlqBaseMask != 31 {
		t.Errorf("vlqBaseMask should be 31, got %d", vlqBaseMask)
	}

	if vlqContinuationBit != 32 {
		t.Errorf("vlqContinuationBit should be 32, got %d", vlqContinuationBit)
	}
}

func TestEncodeVLQLargeValues(t *testing.T) {
	// Test encoding of larger values that require multiple VLQ segments
	largeValues := []int{
		100000, -100000,
		1000000, -1000000,
	}

	for _, val := range largeValues {
		encoded := EncodeVLQ(val)

		// Large values should require multiple characters
		if len(encoded) <= 2 {
			t.Errorf("EncodeVLQ(%d) produced unexpectedly short encoding: '%s'", val, encoded)
		}

		// Verify round-trip
		decoded, _ := DecodeVLQ(encoded)
		if decoded != val {
			t.Errorf("Large value round-trip failed for %d: got %d", val, decoded)
		}
	}
}
