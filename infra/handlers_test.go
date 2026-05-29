package infra

import "testing"

func TestBuiltinHandlers_Echo(t *testing.T) {
	r := BuiltinHandlers()
	out, err := r.Dispatch("echo", map[string]any{"x": 42})
	if err != nil {
		t.Fatal(err)
	}
	if out["x"] != 42 {
		t.Errorf("echo returned %v", out)
	}
}

func TestBuiltinHandlers_Math(t *testing.T) {
	r := BuiltinHandlers()
	cases := []struct {
		op   string
		a, b float64
		want float64
	}{
		{"add", 2, 3, 5}, {"sub", 5, 2, 3}, {"mul", 4, 3, 12}, {"div", 9, 3, 3}, {"pow", 2, 10, 1024},
	}
	for _, c := range cases {
		out, err := r.Dispatch("math", map[string]any{"op": c.op, "a": c.a, "b": c.b})
		if err != nil {
			t.Fatalf("%s: %v", c.op, err)
		}
		if out["value"] != c.want {
			t.Errorf("math %s: got %v want %v", c.op, out["value"], c.want)
		}
	}
}

func TestBuiltinHandlers_StoreRoundTrip(t *testing.T) {
	r := BuiltinHandlers()
	if _, err := r.Dispatch("store.set", map[string]any{"key": "k", "value": "v"}); err != nil {
		t.Fatal(err)
	}
	out, err := r.Dispatch("store.get", map[string]any{"key": "k"})
	if err != nil {
		t.Fatal(err)
	}
	if out["value"] != "v" {
		t.Errorf("store.get returned %v", out)
	}
}

func TestBuiltinHandlers_TimeAndUUID(t *testing.T) {
	r := BuiltinHandlers()
	if out, err := r.Dispatch("time", nil); err != nil || out["time"] == "" {
		t.Errorf("time: out=%v err=%v", out, err)
	}
	if out, err := r.Dispatch("uuid", nil); err != nil || out["value"] == "" {
		t.Errorf("uuid: out=%v err=%v", out, err)
	}
}

func TestBuiltinHandlers_Unknown(t *testing.T) {
	r := BuiltinHandlers()
	if _, err := r.Dispatch("nope", nil); err == nil {
		t.Error("expected error for unknown handler")
	}
}
