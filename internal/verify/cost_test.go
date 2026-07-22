package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
)

// TestIsolatedCostGate runs the built binary in isolated subprocesses under
// /usr/bin/time and asserts the reviewed absolute budgets for wall time and
// peak RSS, three samples per corpus. Budgets were frozen from repeated clean
// calibration runs (webshop ~0.8s/~205MiB, textindex ~0.8s/~195MiB, self
// ~1.8s/~400MiB) with headroom that still fails the pre-partition regression
// class (2.5s/620MiB and 4.2s/1.15GiB respectively). Semantic scope counts
// are asserted by deterministic unit gates, not here.
func TestIsolatedCostGate(t *testing.T) {
	if testing.Short() {
		t.Skip("isolated cost gate skipped in -short")
	}
	timeBinary, err := exec.LookPath("/usr/bin/time")
	if err != nil {
		t.Skip("/usr/bin/time unavailable")
	}
	root := repoRoot(t)
	binary := filepath.Join(t.TempDir(), "gotots-gate")
	build := exec.Command("go", "build", "-o", binary, "./cmd/gotots")
	build.Dir = root
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build: %v\n%s", err, out)
	}
	type budget struct {
		dir        string
		wallLimit  float64 // seconds
		rssLimitKB int
	}
	budgets := map[string]budget{
		"webshop":   {filepath.Join(root, "testdata", "projects", "webshop"), 2.5, 450 * 1024},
		"textindex": {filepath.Join(root, "testdata", "projects", "textindex"), 2.5, 450 * 1024},
		"self":      {root, 5.0, 800 * 1024},
	}
	pattern := regexp.MustCompile(`wall=([0-9.]+) rss=([0-9]+)`)
	for name, b := range budgets {
		// Produce the manifest artifact once per corpus (the gate run), then
		// measure the ordinary consuming compilation in isolation.
		manifest := filepath.Join(t.TempDir(), name+"-manifest.json")
		produce := exec.Command(binary, "audit", "catalog", "-contract", "default@v1", "-dir", b.dir, "-o", manifest)
		produce.Env = os.Environ()
		produceOut, err := produce.CombinedOutput()
		if err != nil {
			t.Fatalf("%s manifest production: %v\n%s", name, err, produceOut)
		}
		digestMatch := regexp.MustCompile(`certifiedDigest=([0-9a-f]{64})`).FindStringSubmatch(string(produceOut))
		if digestMatch == nil {
			t.Fatalf("%s: no certified digest in production output:\n%s", name, produceOut)
		}
		for sample := 1; sample <= 3; sample++ {
			cmd := exec.Command(timeBinary, "-f", "wall=%e rss=%M", binary,
				"inspect", "constructs", "-contract", "default@v1",
				"-manifest", manifest, "-manifest-digest", digestMatch[1], "-dir", b.dir)
			cmd.Env = os.Environ()
			out, err := cmd.CombinedOutput()
			if err != nil {
				t.Fatalf("%s sample %d: %v\n%s", name, sample, err, out)
			}
			match := pattern.FindStringSubmatch(string(out))
			if match == nil {
				t.Fatalf("%s sample %d: no measurement in output", name, sample)
			}
			wall, _ := strconv.ParseFloat(match[1], 64)
			rss, _ := strconv.Atoi(match[2])
			t.Logf("%s sample %d: wall=%.2fs rss=%dKiB", name, sample, wall, rss)
			if wall > b.wallLimit {
				t.Errorf("%s sample %d wall %.2fs exceeds reviewed budget %.2fs", name, sample, wall, b.wallLimit)
			}
			if rss > b.rssLimitKB {
				t.Errorf("%s sample %d peak RSS %dKiB exceeds reviewed budget %dKiB (pre-partition class regression)",
					name, sample, rss, b.rssLimitKB)
			}
		}
	}
	_ = fmt.Sprint
}
