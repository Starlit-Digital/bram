package structured

import (
	"bytes"
	"encoding/json"
	"github.com/cshaiku/bram/internal/appinfo"
	"testing"
)

func TestInterchange(t *testing.T) {
	value := map[string]any{"rows": []any{map[string]any{"title": "00123", "name": "雪|true", "count": int64(9223372036854775807), "empty": nil}}}
	original, _ := json.Marshal(value)
	for _, format := range []string{"json", "gcf", "auto"} {
		wire, err := Marshal(value, format)
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := JSON(wire, 1<<20)
		if err != nil {
			t.Fatal(err)
		}
		var a, b any
		da := json.NewDecoder(bytes.NewReader(original))
		da.UseNumber()
		_ = da.Decode(&a)
		db := json.NewDecoder(bytes.NewReader(decoded))
		db.UseNumber()
		_ = db.Decode(&b)
		aa, _ := json.Marshal(a)
		bb, _ := json.Marshal(b)
		if !bytes.Equal(aa, bb) {
			t.Fatalf("lost value: %s", decoded)
		}
	}
	wire, _ := Marshal(map[string]any{}, "auto")
	if string(wire) != "{}" {
		t.Fatal("small object should stay JSON")
	}
	for _, bad := range []string{"GCF tool=x\n", "{} trailing", "\xff"} {
		if _, err := JSON([]byte(bad), 100); err == nil {
			t.Fatal("accepted invalid input")
		}
	}
	if _, err := JSON([]byte("{}"), 1); err == nil {
		t.Fatal("accepted oversize input")
	}
	if _, err := Marshal(uint64(^uint64(0)), "gcf"); err == nil {
		t.Fatal("accepted out-of-domain integer")
	}
	if _, err := Marshal(uint64(^uint64(0)), "auto"); err != nil {
		t.Fatal("auto must preserve JSON-only integer")
	}
}
func TestPath(t *testing.T) {
	if ReportPath("evidence.auto", []byte("GCF\n")) != "evidence.gcf" {
		t.Fatal("wrong actual suffix")
	}
}
func TestCommands(t *testing.T) {
	var out, errout bytes.Buffer
	handled, code := Command([]string{"capabilities"}, "test-tool", &out, &errout)
	if !handled || code != 0 || !bytes.Contains(out.Bytes(), []byte("data encode")) {
		t.Fatalf("discovery: %s %s", out.Bytes(), errout.Bytes())
	}
	var discovery map[string]any
	if err := json.Unmarshal(out.Bytes(), &discovery); err != nil || discovery["version"] != appinfo.Version {
		t.Fatalf("capabilities version must match app version: %s", out.Bytes())
	}
	out.Reset()
	errout.Reset()
	handled, code = Command([]string{"data", "encode", "-", "--format", "bad"}, "test-tool", &out, &errout)
	if !handled || code == 0 || !bytes.Contains(errout.Bytes(), []byte(`"status":"error"`)) {
		t.Fatal("invalid format must fail before reading input")
	}
}
