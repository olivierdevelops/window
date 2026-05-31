package infra

import (
	"strings"
	"testing"
)

func TestExpandControlFlowFor(t *testing.T) {
	src := `<ul>
{#for label in Home, About, Contact}
  <li><a href="#">"{{ label }}"</a></li>
{/for}
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
	if strings.Contains(out, "{#for") || strings.Contains(out, "{{ label }}") {
		t.Errorf("control tags / vars leaked\n%s", out)
	}
}

func TestExpandControlFlowIfElse(t *testing.T) {
	src := `{#if admin == admin}
  <p>"welcome boss"</p>
{#else}
  <p>"hello user"</p>
{/if}`
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

func TestExpandControlFlowElif(t *testing.T) {
	src := `{#for role in admin, editor, guest}
{#if role == admin}
  <p>"{{ role }}: full"</p>
{#elif role == editor}
  <p>"{{ role }}: edit"</p>
{#else}
  <p>"{{ role }}: view"</p>
{/if}
{/for}`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, want := range []string{`"admin: full"`, `"editor: edit"`, `"guest: view"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestExpandControlFlowIfIn(t *testing.T) {
	src := `{#if b in a, b, c}
  <p>"yes"</p>
{/if}`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"yes"`) {
		t.Errorf("membership match missing\n%s", out)
	}
}

func TestExpandControlFlowMatch(t *testing.T) {
	src := `{#match ok}
  {#case ok}
    <p>"all good"</p>
  {#case err}
    <p>"broken"</p>
  {#default}
    <p>"unknown"</p>
{/match}`
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

func TestExpandControlFlowMatchDefault(t *testing.T) {
	src := `{#match zzz}
  {#case ok}
    <p>"all good"</p>
  {#default}
    <p>"unknown"</p>
{/match}`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	if !strings.Contains(out, `"unknown"`) {
		t.Errorf("default branch missing\n%s", out)
	}
}

func TestExpandControlFlowNested(t *testing.T) {
	// {#if} inside {#for} sees the loop variable.
	src := `{#for role in admin, guest}
{#if role == admin}
  <p>"{{ role }} can edit"</p>
{#else}
  <p>"{{ role }} is read-only"</p>
{/if}
{/for}`
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

func TestExpandControlFlowNestedMatch(t *testing.T) {
	// {#match} inside {#for} resolves the loop variable per item.
	src := `{#for role in admin, editor, guest}
{#match role}
  {#case admin}
    <p>"full"</p>
  {#case editor}
    <p>"edit"</p>
  {#default}
    <p>"view"</p>
{/match}
{/for}`
	out, err := ExpandControlFlow(src)
	if err != nil {
		t.Fatalf("expand: %v", err)
	}
	for _, want := range []string{`"full"`, `"edit"`, `"view"`} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q\n%s", want, out)
		}
	}
}

func TestExpandControlFlowLeavesPlainText(t *testing.T) {
	src := `<app title="X" width="1" height="1">
  <p>"a {#for} in quotes stays"</p>
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
	if _, err := ExpandControlFlow("{#for x in a}\n  <li>\"{{ x }}\"</li>"); err == nil {
		t.Fatal("expected error for unclosed {#for}")
	}
}

func TestExpandControlFlowBadCondition(t *testing.T) {
	if _, err := ExpandControlFlow("{#if role}\n  <p>\"x\"</p>\n{/if}"); err == nil {
		t.Fatal("expected error for a condition without == or in")
	}
}
