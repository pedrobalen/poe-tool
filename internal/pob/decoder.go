package pob

import (
	"bytes"
	"compress/zlib"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"strings"
)

// maxDecompressed caps zlib output to guard against a malicious or corrupt
// code inflating to an unbounded size. Real exports are well under this.
const maxDecompressed = 32 << 20 // 32 MiB

// Decode turns a raw Path of Building export code into its XML bytes.
//
// The export is Base64 (URL-safe alphabet, padding optional) wrapping a
// zlib-compressed XML document. Both the standard and URL-safe alphabets are
// accepted because different sources normalize them differently.
func Decode(code string) ([]byte, error) {
	code = strings.TrimSpace(code)
	if code == "" {
		return nil, errors.New("pob: empty build code")
	}

	compressed, err := decodeBase64(code)
	if err != nil {
		return nil, fmt.Errorf("pob: decoding base64: %w", err)
	}

	xml, err := inflate(compressed)
	if err != nil {
		return nil, fmt.Errorf("pob: decompressing zlib: %w", err)
	}
	if len(xml) == 0 {
		return nil, errors.New("pob: decompressed content is empty")
	}

	return xml, nil
}

// decodeBase64 tolerates the URL-safe alphabet, the standard alphabet, and
// missing padding, trying the most likely form first.
func decodeBase64(code string) ([]byte, error) {
	code = strings.Map(dropWhitespace, code)

	encodings := []*base64.Encoding{
		base64.RawURLEncoding,
		base64.URLEncoding,
		base64.RawStdEncoding,
		base64.StdEncoding,
	}

	var lastErr error
	for _, enc := range encodings {
		data, err := enc.DecodeString(code)
		if err == nil {
			return data, nil
		}
		lastErr = err
	}

	return nil, lastErr
}

// inflate decompresses zlib data with a hard size ceiling.
func inflate(data []byte) ([]byte, error) {
	r, err := zlib.NewReader(bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	defer r.Close()

	limited := io.LimitReader(r, maxDecompressed+1)

	out, err := io.ReadAll(limited)
	if err != nil {
		return nil, err
	}
	if len(out) > maxDecompressed {
		return nil, fmt.Errorf("pob: decompressed content exceeds %d bytes", maxDecompressed)
	}

	return out, nil
}

func dropWhitespace(r rune) rune {
	switch r {
	case ' ', '\t', '\n', '\r':
		return -1
	default:
		return r
	}
}
