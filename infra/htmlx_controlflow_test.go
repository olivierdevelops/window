package infra

import (
	"strings"
	"testing"
)

func TestExpandControlFlowFor(t *testing.T) {
	src := `<ul>
<for each="Home, About, Contact" as="label">
  <li><a href="#">"{label}"</a></li>
</for>
</ul>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, want := range []string{
		`<li><a href="#">"Home"</a></li>`,
		`<li><a href="#">"About"</a></li>`,
		`<li><a href="#">"Contact"</a></li>`,
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n---\n%s", want, out)
		}
	}
	if strings.Contains(out, "<for") || strings.Contains(out, "{label}") {
		t.Errorf("control tags / vars leaked\n%s", out)
	}
}

func TestExpandControlFlowIfElse(t *testing.T) {
	src := `<if value="admin" is="admin">
  <p>"welcome boss"</p>
<else>
  <p>"hello user"</p>
</if>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"welcome boss"`) {
		t.Errorf("then branch missing\n%s", out)
	}
	if strings.Contains(out, `"hello user"`) {
		t.Errorf("else branch should not render\n%s", out)
	}
}

func TestExpandControlFlowIfIn(t *testing.T) {
	src := `<if value="b" in="a, b, c">
  <p>"yes"</p>
</if>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"yes"`) {
		t.Errorf("membership match missing\n%s", out)
	}
}

func TestExpandControlFlowSwitch(t *testing.T) {
	src := `<switch value="ok">
  <case is="ok">
    <p>"all good"</p>
  </case>
  <case is="err">
    <p>"broken"</p>
  </case>
  <default>
    <p>"unknown"</p>
  </default>
</switch>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"all good"`) {
		t.Errorf("matching case missing\n%s", out)
	}
	if strings.Contains(out, `"broken"`) || strings.Contains(out, `"unknown"`) {
		t.Errorf("non-matching branches rendered\n%s", out)
	}
}

func TestExpandControlFlowSwitchDefault(t *testing.T) {
	src := `<switch value="zzz">
  <case is="ok">
    <p>"all good"</p>
  </case>
  <default>
    <p>"unknown"</p>
  </default>
</switch>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"unknown"`) {
		t.Errorf("default branch missing\n%s", out)
	}
}

func TestExpandControlFlowNested(t *testing.T) {
	// <if> inside <for> sees the loop variable.
	src := `<for each="admin, guest" as="role">
<if value="{role}" is="admin">
  <p>"{role} can edit"</p>
<else>
  <p>"{role} is read-only"</p>
</if>
</for>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"admin can edit"`) {
		t.Errorf("nested then missing\n%s", out)
	}
	if !strings.Contains(out, `"guest is read-only"`) {
		t.Errorf("nested else missing\n%s", out)
	}
}

func TestExpandControlFlowLeavesPlainText(t *testing.T) {
	src := `<app title="X" width="1" height="1">
  <p>"a <for> in quotes stays"</p>
</app>`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if out != src {
		t.Errorf("plain source changed\n--- got ---\n%s\n--- want ---\n%s", out, src)
	}
}

func TestExpandControlFlowUnclosed(t *testing.T) {
	if _, err := ExpandControlFlow("<for each=\"a\" as=\"x\">\n  <li>\"{x}\"</li>"); err == nil {
		t.Fatal("expected error for unclosed <for>")
	}
}
