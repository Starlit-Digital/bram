package structured

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/cshaiku/bram/internal/appinfo"
	"io"
	"os"
)

// Command handles read-only common commands before tool-specific execution.
func Command(args []string, tool string, stdout, stderr io.Writer) (bool, int) {
	return CommandWithInput(args, tool, os.Stdin, stdout, stderr)
}
func CommandWithInput(args []string, tool string, stdin io.Reader, stdout, stderr io.Writer) (bool, int) {
	if len(args) == 0 || (args[0] != "capabilities" && args[0] != "data") {
		return false, 0
	}
	err := command(args, tool, stdin, stdout)
	if err != nil {
		body, _ := json.Marshal(map[string]any{"schemaVersion": 1, "status": "error", "message": err.Error()})
		fmt.Fprintln(stderr, string(body))
		return true, 1
	}
	return true, 0
}
func command(args []string, tool string, stdin io.Reader, out io.Writer) error {
	capabilities := args[0] == "capabilities"
	args = args[1:]
	action := "capabilities"
	if !capabilities {
		if len(args) == 0 {
			return fmt.Errorf("usage: data encode|decode|stats FILE|- [--format json|gcf|auto]")
		}
		action = args[0]
		args = args[1:]
	}
	format := "json"
	if action == "encode" {
		format = "gcf"
	}
	for i, arg := range args {
		if arg == "--format" {
			if i+1 >= len(args) {
				return fmt.Errorf("--format requires json, gcf or auto")
			}
			format = args[i+1]
			args = append(args[:i:i], args[i+2:]...)
			break
		}
	}
	if !ValidFormat(format) {
		return fmt.Errorf("--format requires json, gcf or auto")
	}
	var value any
	if capabilities {
		if len(args) != 0 {
			return fmt.Errorf("capabilities accepts only --format")
		}
		value = map[string]any{"schemaVersion": 1, "tool": tool, "formats": []string{"json", "gcf", "auto"}, "commands": []string{"capabilities", "data encode", "data decode", "data stats"}, "maximumBytes": MaxBytes, "dataCommandsReadOnly": true, "structuredInterfaces": []string{"CLI results: text/json/gcf/auto", "HTTP JSON/GCF bodies; application/gcf Accept or ?format=", "JSON/GCF peer configuration"}, "planExecution": "requires explicit tool-specific arguments", "profile": "generic snapshot only", "version": appinfo.Version}
	} else {
		if len(args) != 1 || (action != "encode" && action != "decode" && action != "stats") {
			return fmt.Errorf("usage: data encode|decode|stats FILE|- [--format json|gcf|auto]")
		}
		var data []byte
		var err error
		if args[0] == "-" {
			data, err = io.ReadAll(io.LimitReader(stdin, MaxBytes+1))
		} else {
			data, err = ReadFile(args[0], MaxBytes)
		}
		if err != nil {
			return err
		}
		data, err = JSON(data, MaxBytes)
		if err != nil {
			return err
		}
		decoder := json.NewDecoder(bytes.NewReader(data))
		decoder.UseNumber()
		if err = decoder.Decode(&value); err != nil {
			return err
		}
		if action == "stats" {
			a, err := Marshal(value, "json")
			if err != nil {
				return err
			}
			b, err := Marshal(value, "gcf")
			if err != nil {
				return err
			}
			selected := "json"
			if len(b) < len(a) {
				selected = "gcf"
			}
			value = map[string]any{"schemaVersion": 1, "jsonBytes": len(a), "gcfBytes": len(b), "selected": selected}
		}
	}
	body, err := Marshal(value, format)
	if err != nil {
		return err
	}
	_, err = out.Write(body)
	if err == nil && !bytes.HasSuffix(body, []byte("\n")) {
		_, err = fmt.Fprintln(out)
	}
	return err
}
