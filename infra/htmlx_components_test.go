package infra

import (
	"strings"
	"testing"
)

func TestRewriteHTMLXComponents(t *testing.T) {
	src := `<component name="card" props="title">
  <section class="card">
    <h3>{{ title }}</h3>
    <slot></slot>
  </section>
</component>

<app title="X" width="400" height="300">
  <card title="Hi">"body"</card>
</app>`

	out, err := RewriteHTMLXComponents(src)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}

	for _, want := range []string{
		"define card",
		`arg literal "card"`,
		`arg capture title raw`,
		`block_close_seq "</" "card" ">"`,
		"${escapeHtml title}", // {title} prop interpolation
		"${body}",             // <slot> became the body slot
	} {
		if !strings.Contains(out, want) {
			t.Errorf("rewritten source missing %q\n---\n%s", want, out)
		}
	}
	// The <app> usage must be preserved untouched.
	if !strings.Contains(out, `<card title="Hi">"body"</card>`) {
		t.Error("component usage was not preserved in the rewritten source")
	}
}

func TestRewriteHTMLXComponentsVoid(t *testing.T) {
	src := `<component name="avatar" props="name" void>
  <img src="?n={{ name }}" />
</component>
<app title="X" width="1" height="1"><avatar name="Ada" /></app>`

	out, err := RewriteHTMLXComponents(src)
	if err != nil {
		t.Fatalf("rewrite: %v", err)
	}
	if !strings.Contains(out, `arg literal "/"`) {
		t.Errorf("void component should emit a self-closing `/` literal\n%s", out)
	}
	if strings.Contains(out, "block_close_seq") {
		t.Errorf("void component must not emit a block closer\n%s", out)
	}
}

func TestRewriteHTMLXComponentsIgnoresCommentsAndText(t *testing.T) {
	// A <component> mentioned in a # comment or a "quoted" text node must NOT be
	// treated as a definition.
	src := `# see <component> docs
<app title="X" width="1" height="1">
  <p>"Use "<code>"<component>"</code>" here."</p>
</app>`

	out, err := RewriteHTMLXComponents(src)
	if err != nil {
		t.Fatalf("rewrite should be a no-op, got: %v", err)
	}
	if strings.Contains(out, "define") {
		t.Errorf("no component should have been extracted\n%s", out)
	}
}

func TestRewriteHTMLXComponentsRequiresName(t *testing.T) {
	src := "<component props=\"x\">\n  <b>{{ x }}</b>\n</component>"
	if _, err := RewriteHTMLXComponents(src); err == nil {
		t.Fatal("expected an error for a <component> with no name")
	}
}
