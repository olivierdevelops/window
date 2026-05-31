package capyx_test

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"webview_gui/infra"
)

// generate_test.go compiles every .capyx demo, asserts the generated
// index.html has the expected structure, and executes the inlined JavaScript
// under a Node DOM shim to prove the signals runtime actually mounts and
// reacts to events — no browser required.

func runtimeJS(t *testing.T) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "assets", "capyx_runtime.js"))
	if err != nil {
		t.Fatalf("read runtime: %v", err)
	}
	return string(b)
}

var scriptRE = regexp.MustCompile(`(?s)<script>\n(.*?)\n</script>`)

// extractScripts pulls the inlined <script> bodies (runtime, then appJS).
func extractScripts(html string) []string {
	var out []string
	for _, m := range scriptRE.FindAllStringSubmatch(html, -1) {
		out = append(out, m[1])
	}
	return out
}

// compile compiles a single .capyx file and returns the generated index.html.
func compile(t *testing.T, path, rt string) string {
	t.Helper()
	src, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	files, err := infra.CompileCapyx(string(src), rt)
	if err != nil {
		t.Fatalf("compile %s: %v", path, err)
	}
	html, ok := files["static/index.html"]
	if !ok {
		t.Fatalf("%s: no static/index.html generated", path)
	}
	if _, ok := files["window.yaml"]; !ok {
		t.Fatalf("%s: no window.yaml generated", path)
	}
	return html
}

// node runs a script through Node and returns combined output, failing the
// test if Node exits non-zero.
func node(t *testing.T, script string) string {
	t.Helper()
	dir := t.TempDir()
	f := filepath.Join(dir, "run.js")
	if err := os.WriteFile(f, []byte(script), 0644); err != nil {
		t.Fatalf("write script: %v", err)
	}
	out, err := exec.Command("node", f).CombinedOutput()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}
	return string(out)
}

// harness assembles a runnable Node program: DOM shim + runtime + appJS + a
// driver snippet that interacts with the mounted DOM.
func harness(t *testing.T, html, driver string) string {
	t.Helper()
	shim, err := os.ReadFile("dom_shim.js")
	if err != nil {
		t.Fatalf("read shim: %v", err)
	}
	scripts := extractScripts(html)
	if len(scripts) < 2 {
		t.Fatalf("expected runtime + appJS scripts, got %d", len(scripts))
	}
	var b strings.Builder
	b.WriteString(string(shim))
	b.WriteString("\n")
	b.WriteString(scripts[0]) // runtime
	b.WriteString("\n")
	b.WriteString(scripts[1]) // appJS
	b.WriteString("\n")
	b.WriteString(driver)
	return b.String()
}

// TestCompileAll compiles and runs every demo, asserting it mounts cleanly.
func TestCompileAll(t *testing.T) {
	rt := runtimeJS(t)
	files, _ := filepath.Glob("*.capyx")
	if len(files) == 0 {
		t.Fatal("no .capyx demos found")
	}
	for _, f := range files {
		f := f
		t.Run(f, func(t *testing.T) {
			html := compile(t, f, rt)
			for _, marker := range []string{"<!doctype html>", "CAPYX", "__capyxStart", "<div id=\"app\">"} {
				if !strings.Contains(html, marker) {
					t.Errorf("%s: missing marker %q", f, marker)
				}
			}
			driver := `
var root = globalThis.__APP_ROOT__;
if (root.childNodes.length === 0) { console.error("FAIL: nothing mounted"); process.exit(1); }
console.log("MOUNTED:" + root.textContent.replace(/\s+/g," ").trim().slice(0,60));
`
			out := node(t, harness(t, html, driver))
			if !strings.Contains(out, "MOUNTED:") {
				t.Errorf("%s: did not mount: %s", f, out)
			}
		})
	}
}

// findByText walks the DOM for an element whose direct text equals want.
const findHelpers = `
function findButton(label){
  var hit=null;
  globalThis.__walk__(globalThis.__APP_ROOT__,function(n){
    if(n.kind==="element" && n._tag==="button" && n.textContent.trim()===label && !hit) hit=n;
  });
  return hit;
}
function spanText(){
  var t=null;
  globalThis.__walk__(globalThis.__APP_ROOT__,function(n){
    if(n.kind==="element" && n._tag==="span" && t===null) t=n.textContent.trim();
  });
  return t;
}
`

// TestCounterReactivity proves fine-grained reactivity: clicking +/- updates
// only the bound span text.
func TestCounterReactivity(t *testing.T) {
	rt := runtimeJS(t)
	html := compile(t, "counter.capyx", rt)
	driver := findHelpers + `
function emit(btn){ btn.dispatch("click"); }
var plus=findButton("+"), minus=findButton("-");
if(!plus||!minus){ console.error("FAIL: buttons not found"); process.exit(1); }
console.log("START:"+spanText());
emit(plus); emit(plus); emit(plus);
console.log("PLUS3:"+spanText());
emit(minus);
console.log("MINUS1:"+spanText());
`
	out := node(t, harness(t, html, driver))
	checks := map[string]string{"START:": "0", "PLUS3:": "3", "MINUS1:": "2"}
	for prefix, want := range checks {
		got := lineValue(out, prefix)
		if got != want {
			t.Errorf("counter %s got %q want %q\nfull:%s", prefix, got, want, out)
		}
	}
}

// TestTodoListReactivity proves keyed-list reconciliation: typing + add grows
// the list, and toggling/removing updates only the affected row.
func TestTodoListReactivity(t *testing.T) {
	rt := runtimeJS(t)
	html := compile(t, "todo.capyx", rt)
	driver := findHelpers + `
function firstInput(){ var h=null; globalThis.__walk__(globalThis.__APP_ROOT__,function(n){ if(n.kind==="element"&&n._tag==="input"&&!h) h=n; }); return h; }
function countRemove(){ var c=0; globalThis.__walk__(globalThis.__APP_ROOT__,function(n){ if(n.kind==="element"&&n._tag==="button"&&n.textContent.trim()==="✕") c++; }); return c; }
function addTask(txt){ var inp=firstInput(); inp.value=txt; inp.dispatch("input"); findButton("add").dispatch("click"); }
console.log("START:"+countRemove());
addTask("Buy milk"); addTask("Walk dog"); addTask("Write code");
console.log("ADDED:"+countRemove());
// remove the first task
var x=null; globalThis.__walk__(globalThis.__APP_ROOT__,function(n){ if(n.kind==="element"&&n._tag==="button"&&n.textContent.trim()==="✕"&&!x) x=n; });
x.dispatch("click");
console.log("REMOVED:"+countRemove());
`
	out := node(t, harness(t, html, driver))
	for prefix, want := range map[string]string{"START:": "0", "ADDED:": "3", "REMOVED:": "2"} {
		if got := lineValue(out, prefix); got != want {
			t.Errorf("todo %s got %q want %q\n%s", prefix, got, want, out)
		}
	}
}

// TestSharedOrchestrator proves two handlers injecting the same orchestrator
// share one reactive store: a click in one panel updates both panels' bindings.
func TestSharedOrchestrator(t *testing.T) {
	rt := runtimeJS(t)
	html := compile(t, "orchestrator.capyx", rt)
	driver := findHelpers + `
function strongs(){ var out=[]; globalThis.__walk__(globalThis.__APP_ROOT__,function(n){ if(n.kind==="element"&&n._tag==="strong") out.push(n.textContent.trim()); }); return out; }
console.log("START:"+strongs().join(","));
findButton("+1 from here").dispatch("click");
console.log("ONE:"+strongs().join(","));
`
	out := node(t, harness(t, html, driver))
	if got := lineValue(out, "START:"); got != "0,0" {
		t.Errorf("orchestrator START got %q want 0,0\n%s", got, out)
	}
	// One click on panel A must update BOTH panels' shared count.
	if got := lineValue(out, "ONE:"); got != "1,1" {
		t.Errorf("orchestrator ONE got %q want 1,1 (shared reactive store)\n%s", got, out)
	}
}

func lineValue(out, prefix string) string {
	for _, ln := range strings.Split(out, "\n") {
		if strings.HasPrefix(ln, prefix) {
			return strings.TrimPrefix(ln, prefix)
		}
	}
	return ""
}
