// Package structured provides bounded, exact JSON/generic-GCF interchange.
package structured

import (
	"bytes"
	"encoding/json"
	"fmt"
	gcf "github.com/blackwell-systems/gcf-go"
	"io"
	"os"
	"strings"
	"unicode/utf8"
)

const MaxBytes = 64 << 20

func ValidFormat(format string) bool { return format == "json" || format == "gcf" || format == "auto" }
func Marshal(value any, format string) ([]byte, error) {
	if !ValidFormat(format) {
		return nil, fmt.Errorf("format must be json, gcf or auto")
	}
	data, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	if len(data) > MaxBytes {
		return nil, fmt.Errorf("structured output too large")
	}
	if format == "json" {
		return data, nil
	}
	ordered, err := gcf.ParseJSONOrdered(data)
	if err != nil {
		if format == "auto" {
			return data, nil
		}
		return nil, err
	}
	wire, err := gcf.EncodeGenericChecked(ordered, gcf.GenericOptions{NoFlatten: true})
	if err == nil && len(wire) > MaxBytes {
		err = fmt.Errorf("GCF output too large")
	}
	if format == "gcf" {
		return []byte(wire), err
	}
	if err == nil && len(wire) < len(data) {
		return []byte(wire), nil
	}
	return data, nil
}
func JSON(input []byte, limit int) ([]byte, error) {
	if len(input) > limit || !utf8.Valid(input) {
		return nil, fmt.Errorf("invalid UTF-8 or oversized structured input")
	}
	trimmed := bytes.TrimSpace(input)
	if bytes.HasPrefix(trimmed, []byte("GCF")) {
		header, _, _ := strings.Cut(string(trimmed), "\n")
		generic := false
		stateful := false
		for _, field := range strings.Fields(header) {
			if field == "profile=generic" {
				generic = true
			}
			for _, prefix := range []string{"delta=", "unchanged=", "session=", "base_root=", "new_root="} {
				if strings.HasPrefix(field, prefix) {
					stateful = true
				}
			}
		}
		if !generic || stateful {
			return nil, fmt.Errorf("expected a complete generic GCF snapshot")
		}
		value, err := gcf.DecodeGeneric(string(trimmed))
		if err != nil {
			return nil, err
		}
		data, err := json.Marshal(value)
		if err != nil {
			return nil, err
		}
		if len(data) > limit {
			return nil, fmt.Errorf("decoded input too large")
		}
		return data, nil
	}
	if !json.Valid(input) {
		return nil, fmt.Errorf("expected one JSON value")
	}
	return input, nil
}
func ReportPath(path string, body []byte) string {
	extension := ".json"
	if bytes.HasPrefix(body, []byte("GCF")) {
		extension = ".gcf"
	}
	for _, suffix := range []string{".json", ".gcf", ".auto", ".vdata"} {
		if strings.HasSuffix(strings.ToLower(path), suffix) {
			return path[:len(path)-len(suffix)] + extension
		}
	}
	return path + extension
}

// ReadFile bounds allocation even if the file grows after admission.
func ReadFile(path string, limit int) ([]byte, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return nil, err
	}
	if !info.Mode().IsRegular() || info.Size() > int64(limit) {
		return nil, fmt.Errorf("input must be a bounded regular file")
	}
	bytes, err := io.ReadAll(io.LimitReader(file, int64(limit)+1))
	if err != nil {
		return nil, err
	}
	if len(bytes) > limit {
		return nil, fmt.Errorf("input exceeds byte limit")
	}
	return bytes, nil
}
