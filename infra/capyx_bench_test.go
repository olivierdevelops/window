package infra

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"webview_gui/assets"
)

// capyx_bench_test.go — smoke test for the interactive Test Bench. It loads the
// bench's scripts (runtime + harness + kernel + bench UI) under the Node DOM
// shim and asserts the bench chrome renders without error, a component is
// auto-selected into the isolated preview, and the kernel still drives a
// scenario afterwards. (The visual layout is exercised by hand in the real
// webview; this guards against the bench JS throwing on load.)
func TestCapyxBenchSmoke(t *testing.T) {
	if _, err := exec.LookPath("node"); err != nil {
		t.Skip("node not on PATH")
	}
	capyxSrc := readDemo(t, "counter.capyx")
	harnessJS, err := CompileCapyxHarnessJS(capyxSrc, string(assets.CapyxRuntimeJS))
	if err != nil {
		t.Fatalf("harness: %v", err)
	}
	shim, err := os.ReadFile(filepath.Join("..", "demos", "capyx", "dom_shim.js"))
	if err != nil {
		t.Fatalf("shim: %v", err)
	}

	var prog strings.Builder
	prog.Write(shim)
	prog.WriteString("\n" + string(assets.CapyxRuntimeJS))
	prog.WriteString("\n" + harnessJS)
	prog.WriteString("\n" + string(assets.CapyxTestkitJS))
	prog.WriteString("\n" + string(assets.CapyxTestbenchJS))
	prog.WriteString(`
var root = globalThis.__APP_ROOT__;
var txt = root.textContent.replace(/\s+/g," ");
console.log("BENCH:" + (txt.indexOf("Test Bench") >= 0));
console.log("COMPS:" + (txt.indexOf("Components") >= 0));
console.log("EVENTS:" + (txt.indexOf("Events") >= 0));
console.log("HASBENCH:" + (!!globalThis.CAPYX_BENCH));
var res = CAPYX_TEST_KIT.runScenario({name:"x",steps:[
  {op:"mount",component:"counter"},
  {op:"click",text:"+"},
  {op:"click",text:"+"},
  {op:"expectState",field:"count",value:2}
]});
console.log("RUN:" + res.ok);
`)

	dir := t.TempDir()
	f := filepath.Join(dir, "bench.js")
	if err := os.WriteFile(f, []byte(prog.String()), 0644); err != nil {
		t.Fatal(err)
	}
	out, err := exec.Command("node", f).CombinedOutput()
	if err != nil {
		t.Fatalf("node failed: %v\n%s", err, out)
	}
	got := string(out)
	for _, want := range []string{"BENCH:true", "COMPS:true", "EVENTS:true", "HASBENCH:true", "RUN:true"} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in bench output:\n%s", want, got)
		}
	}
}

// TestCompileCapyxBenchPage checks the Go bench assembler emits a well-formed
// page: a window.yaml, the bench mount point, and the four inlined scripts
// (runtime, harness, kernel, bench UI) in order.
func TestCompileCapyxBenchPage(t *testing.T) {
	files, err := CompileCapyxBench(readDemo(t, "counter.capyx"),
		string(assets.CapyxRuntimeJS), string(assets.CapyxTestkitJS), string(assets.CapyxTestbenchJS))
	if err != nil {
		t.Fatalf("bench compile: %v", err)
	}
	if _, ok := files["window.yaml"]; !ok {
		t.Fatal("no window.yaml")
	}
	html, ok := files["static/index.html"]
	if !ok {
		t.Fatal("no index.html")
	}
	for _, want := range []string{"capyx-bench", "Test Bench", "CAPYX", "__CAPYX_TEST__", "CAPYX_TEST_KIT", "CAPYX_BENCH"} {
		if !strings.Contains(html, want) {
			t.Errorf("bench page missing %q", want)
		}
	}
	if n := strings.Count(html, "<script>"); n != 4 {
		t.Errorf("expected 4 inlined scripts, got %d", n)
	}
}
