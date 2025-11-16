package sourcemap

// VLQ (Variable Length Quantity) encoding for source maps
// Based on the Source Map v3 specification

const (
	vlqBaseShift       = 5
	vlqBase            = 1 << vlqBaseShift // 32
	vlqBaseMask        = vlqBase - 1       // 31
	vlqContinuationBit = vlqBase           // 32
)

var base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

// EncodeVLQ encodes an integer as a VLQ (Variable Length Quantity) base64 string
func EncodeVLQ(value int) string {
	var result []byte

	// Convert to unsigned and handle sign
	var vlq int
	if value < 0 {
		vlq = ((-value) << 1) | 1
	} else {
		vlq = value << 1
	}

	// Encode as VLQ
	for {
		digit := vlq & vlqBaseMask
		vlq >>= vlqBaseShift

		if vlq > 0 {
			// More digits to come, set continuation bit
			digit |= vlqContinuationBit
		}

		result = append(result, base64Chars[digit])

		if vlq == 0 {
			break
		}
	}

	return string(result)
}

// DecodeVLQ decodes a VLQ base64 string to an integer
// Returns the decoded value and the number of characters consumed
func DecodeVLQ(encoded string) (int, int) {
	var result int
	var shift uint
	var continuation bool
	charsRead := 0

	for i := 0; i < len(encoded); i++ {
		charsRead++
		char := encoded[i]

		// Find digit value
		var digit int
		if char >= 'A' && char <= 'Z' {
			digit = int(char - 'A')
		} else if char >= 'a' && char <= 'z' {
			digit = int(char-'a') + 26
		} else if char >= '0' && char <= '9' {
			digit = int(char-'0') + 52
		} else if char == '+' {
			digit = 62
		} else if char == '/' {
			digit = 63
		}

		continuation = (digit & vlqContinuationBit) != 0
		digit &= vlqBaseMask

		result += digit << shift
		shift += vlqBaseShift

		if !continuation {
			break
		}
	}

	// Handle sign
	if result&1 == 1 {
		result = -(result >> 1)
	} else {
		result = result >> 1
	}

	return result, charsRead
}
