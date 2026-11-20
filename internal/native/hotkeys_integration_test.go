package native

import (
	"os/exec"
	"path/filepath"
	"testing"
)

func TestHotkeysReceiveDispatcherEvents(t *testing.T) {
	binary := filepath.Join(t.TempDir(), "hotkey-route")
	cmd := exec.Command("xcrun", "clang", "-fobjc-arc", "-Wno-incomplete-implementation", "-I.", "testdata/hotkey_route.m", "hotkeys_darwin.m", "-framework", "AppKit", "-framework", "Carbon", "-o", binary)
	if out, e := cmd.CombinedOutput(); e != nil {
		t.Fatalf("compile harness: %v\n%s", e, out)
	}
	if out, e := exec.Command(binary).CombinedOutput(); e != nil {
		t.Fatalf("native event route: %v\n%s", e, out)
	}
}
