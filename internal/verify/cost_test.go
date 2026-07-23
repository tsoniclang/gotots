package verify

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

const stage1CertificationEnvironment = "GOTOTS_CERTIFY_STAGE1"

type stage1CostBudget struct {
	name                  string
	directory             string
	auditWallSeconds      float64
	auditPeakRSSKiB       int
	inspectWallSeconds    float64
	inspectPeakRSSKiB     int
	artifactBytes         int64
	largestShardBytes     int64
	largestPackageRecords int
}

type processMeasurement struct {
	wallSeconds float64
	peakRSSKiB  int
	output      string
}

func TestIsolatedStage1CostGate(t *testing.T) {
	if os.Getenv(stage1CertificationEnvironment) != "1" {
		t.Skip(
			"set GOTOTS_CERTIFY_STAGE1=1 for the fail-closed phase-exit cost gate",
		)
	}
	timeBinary := requiredCostTool(t, "/usr/bin/time")
	timeoutBinary := requiredCostTool(t, "timeout")
	prlimitBinary := requiredCostTool(t, "prlimit")
	root := repoRoot(t)
	binary := filepath.Join(t.TempDir(), "gotots-stage1-gate")
	runBoundedCostCommand(
		t,
		"build",
		root,
		prlimitBinary,
		timeoutBinary,
		120,
		"go",
		"build",
		"-o",
		binary,
		"./cmd/gotots",
	)

	budgets := []stage1CostBudget{
		{
			name: "webshop",
			directory: filepath.Join(
				root,
				"testdata",
				"projects",
				"webshop",
			),
			auditWallSeconds:      25,
			auditPeakRSSKiB:       700 * 1024,
			inspectWallSeconds:    5,
			inspectPeakRSSKiB:     350 * 1024,
			artifactBytes:         12 * 1024 * 1024,
			largestShardBytes:     4 * 1024 * 1024,
			largestPackageRecords: 180000,
		},
		{
			name: "textindex",
			directory: filepath.Join(
				root,
				"testdata",
				"projects",
				"textindex",
			),
			auditWallSeconds:      25,
			auditPeakRSSKiB:       700 * 1024,
			inspectWallSeconds:    5,
			inspectPeakRSSKiB:     350 * 1024,
			artifactBytes:         12 * 1024 * 1024,
			largestShardBytes:     4 * 1024 * 1024,
			largestPackageRecords: 180000,
		},
		{
			name:                  "self",
			directory:             root,
			auditWallSeconds:      40,
			auditPeakRSSKiB:       700 * 1024,
			inspectWallSeconds:    20,
			inspectPeakRSSKiB:     900 * 1024,
			artifactBytes:         20 * 1024 * 1024,
			largestShardBytes:     4 * 1024 * 1024,
			largestPackageRecords: 180000,
		},
	}
	for _, budget := range budgets {
		t.Run(budget.name, func(t *testing.T) {
			runStage1CostCorpus(
				t,
				binary,
				timeBinary,
				timeoutBinary,
				prlimitBinary,
				budget,
			)
		})
	}
}

func runStage1CostCorpus(
	t *testing.T,
	binary string,
	timeBinary string,
	timeoutBinary string,
	prlimitBinary string,
	budget stage1CostBudget,
) {
	t.Helper()
	var certifiedDigest string
	var encodedBytes int64
	var largestShardBytes int64
	var largestPackageRecords int
	for sample := 1; sample <= 3; sample++ {
		artifact := filepath.Join(
			t.TempDir(),
			fmt.Sprintf("%s-%d.gotots", budget.name, sample),
		)
		measurement := measureBoundedCostCommand(
			t,
			fmt.Sprintf("%s-audit-%d", budget.name, sample),
			rootForCostDirectory(t, budget.directory),
			timeBinary,
			prlimitBinary,
			timeoutBinary,
			180,
			binary,
			"audit",
			"catalog",
			"-contract",
			"portable@v1",
			"-dir",
			budget.directory,
			"-o",
			artifact,
		)
		assertMeasurement(
			t,
			budget.name+" audit",
			sample,
			measurement,
			budget.auditWallSeconds,
			budget.auditPeakRSSKiB,
		)
		digest := parseTextField(
			t,
			measurement.output,
			`certifiedDigest=([0-9a-f]{64})`,
		)
		sampleBytes := parseInt64Field(
			t,
			measurement.output,
			`encodedBytes=([0-9]+)`,
		)
		sampleShardBytes := parseInt64Field(
			t,
			measurement.output,
			`largestShardBytes=([0-9]+)`,
		)
		samplePackageRecords := parseIntField(
			t,
			measurement.output,
			`largestPackageRecords=([0-9]+)`,
		)
		packageContexts := parseIntField(
			t,
			measurement.output,
			`packageContexts=([0-9]+)`,
		)
		definitions := parseIntField(
			t,
			measurement.output,
			`definitions=([0-9]+)`,
		)
		assertTailLineCount(
			t,
			measurement.output,
			"provider-production-tail ",
			minimum(packageContexts, 20),
		)
		assertTailLineCount(
			t,
			measurement.output,
			"provider-header-tail ",
			minimum(definitions, 20),
		)
		if sampleBytes > budget.artifactBytes ||
			sampleShardBytes > budget.largestShardBytes ||
			samplePackageRecords > budget.largestPackageRecords {
			t.Fatalf(
				"%s audit sample %d artifact=%d shard=%d records=%d exceeds budgets %d/%d/%d",
				budget.name,
				sample,
				sampleBytes,
				sampleShardBytes,
				samplePackageRecords,
				budget.artifactBytes,
				budget.largestShardBytes,
				budget.largestPackageRecords,
			)
		}
		if sample == 1 {
			certifiedDigest = digest
			encodedBytes = sampleBytes
			largestShardBytes = sampleShardBytes
			largestPackageRecords = samplePackageRecords
		} else if digest != certifiedDigest ||
			sampleBytes != encodedBytes ||
			sampleShardBytes != largestShardBytes ||
			samplePackageRecords != largestPackageRecords {
			t.Fatalf(
				"%s provider production is nondeterministic",
				budget.name,
			)
		}
		if sample == 3 {
			for inspectSample := 1; inspectSample <= 3; inspectSample++ {
				inspect := measureBoundedCostCommand(
					t,
					fmt.Sprintf(
						"%s-inspect-%d",
						budget.name,
						inspectSample,
					),
					rootForCostDirectory(t, budget.directory),
					timeBinary,
					prlimitBinary,
					timeoutBinary,
					90,
					binary,
					"inspect",
					"constructs",
					"-contract",
					"portable@v1",
					"-dir",
					budget.directory,
					"-provider",
					artifact,
					"-provider-digest",
					certifiedDigest,
				)
				assertMeasurement(
					t,
					budget.name+" inspect",
					inspectSample,
					inspect,
					budget.inspectWallSeconds,
					budget.inspectPeakRSSKiB,
				)
				assertBoundedProviderConsumption(
					t,
					budget,
					inspect.output,
				)
			}
		}
	}
	t.Logf(
		"%s provider artifact=%d bytes largest-shard=%d bytes largest-package=%d records",
		budget.name,
		encodedBytes,
		largestShardBytes,
		largestPackageRecords,
	)
}

func requiredCostTool(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf(
			"Stage-1 certification requires %s: %v",
			name,
			err,
		)
	}
	return path
}

func runBoundedCostCommand(
	t *testing.T,
	name string,
	directory string,
	prlimitBinary string,
	timeoutBinary string,
	timeoutSeconds int,
	command string,
	arguments ...string,
) *exec.Cmd {
	t.Helper()
	commandArguments := []string{
		"--as=4294967296",
		"--",
		timeoutBinary,
		"--signal=TERM",
		"--kill-after=10s",
		fmt.Sprintf("%ds", timeoutSeconds),
		command,
	}
	commandArguments = append(commandArguments, arguments...)
	cmd := exec.Command(prlimitBinary, commandArguments...)
	cmd.Dir = directory
	cmd.Env = boundedCostEnvironment()
	logPath := filepath.Join(t.TempDir(), name+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	if err := cmd.Run(); err != nil {
		_ = logFile.Close()
		t.Fatalf(
			"%s failed: %v\n%s",
			name,
			err,
			readBoundedCostLog(t, logPath),
		)
	}
	if err := logFile.Close(); err != nil {
		t.Fatal(err)
	}
	return cmd
}

func measureBoundedCostCommand(
	t *testing.T,
	name string,
	directory string,
	timeBinary string,
	prlimitBinary string,
	timeoutBinary string,
	timeoutSeconds int,
	command string,
	arguments ...string,
) processMeasurement {
	t.Helper()
	timedArguments := []string{
		"-f",
		"GOTOTS_COST wall=%e rss=%M",
		command,
	}
	timedArguments = append(timedArguments, arguments...)
	commandArguments := []string{
		"--as=4294967296",
		"--",
		timeoutBinary,
		"--signal=TERM",
		"--kill-after=10s",
		fmt.Sprintf("%ds", timeoutSeconds),
		timeBinary,
	}
	commandArguments = append(commandArguments, timedArguments...)
	cmd := exec.Command(prlimitBinary, commandArguments...)
	cmd.Dir = directory
	cmd.Env = boundedCostEnvironment()
	logPath := filepath.Join(t.TempDir(), name+".log")
	logFile, err := os.Create(logPath)
	if err != nil {
		t.Fatal(err)
	}
	cmd.Stdout = logFile
	cmd.Stderr = logFile
	runErr := cmd.Run()
	if closeErr := logFile.Close(); closeErr != nil {
		t.Fatal(closeErr)
	}
	output := readBoundedCostLog(t, logPath)
	if runErr != nil {
		t.Fatalf("%s failed: %v\n%s", name, runErr, output)
	}
	return processMeasurement{
		wallSeconds: parseFloatField(
			t,
			output,
			`GOTOTS_COST wall=([0-9.]+)`,
		),
		peakRSSKiB: parseIntField(
			t,
			output,
			`GOTOTS_COST wall=[0-9.]+ rss=([0-9]+)`,
		),
		output: output,
	}
}

func boundedCostEnvironment() []string {
	replacements := map[string]string{
		"GOMEMLIMIT": "768MiB",
		"GOMAXPROCS": "2",
		"GOFLAGS":    "-p=1",
	}
	environment := make([]string, 0, len(os.Environ())+len(replacements))
	for _, entry := range os.Environ() {
		key, _, found := strings.Cut(entry, "=")
		if _, replace := replacements[key]; found && replace {
			continue
		}
		environment = append(environment, entry)
	}
	for _, key := range []string{"GOMEMLIMIT", "GOMAXPROCS", "GOFLAGS"} {
		environment = append(
			environment,
			key+"="+replacements[key],
		)
	}
	return environment
}

func rootForCostDirectory(t *testing.T, directory string) string {
	t.Helper()
	current := directory
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}
		parent := filepath.Dir(current)
		if parent == current {
			t.Fatalf("no go.mod ancestor for %s", directory)
		}
		current = parent
	}
}

func readBoundedCostLog(t *testing.T, path string) string {
	t.Helper()
	stat, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	const maximumLogBytes = 2 * 1024 * 1024
	if stat.Size() > maximumLogBytes {
		t.Fatalf(
			"cost log %s is %d bytes, exceeds %d",
			path,
			stat.Size(),
			maximumLogBytes,
		)
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(raw)
}

func assertMeasurement(
	t *testing.T,
	name string,
	sample int,
	measurement processMeasurement,
	wallLimit float64,
	rssLimit int,
) {
	t.Helper()
	t.Logf(
		"%s sample %d: wall=%.2fs peakRSS=%dKiB",
		name,
		sample,
		measurement.wallSeconds,
		measurement.peakRSSKiB,
	)
	if measurement.wallSeconds > wallLimit ||
		measurement.peakRSSKiB > rssLimit {
		t.Fatalf(
			"%s sample %d exceeds wall/RSS budgets: %.2fs/%dKiB > %.2fs/%dKiB",
			name,
			sample,
			measurement.wallSeconds,
			measurement.peakRSSKiB,
			wallLimit,
			rssLimit,
		)
	}
}

func parseTextField(
	t *testing.T,
	output string,
	pattern string,
) string {
	t.Helper()
	match := regexp.MustCompile(pattern).FindStringSubmatch(output)
	if len(match) != 2 {
		t.Fatalf(
			"output lacks %q:\n%s",
			pattern,
			strings.TrimSpace(output),
		)
	}
	return match[1]
}

func parseIntField(
	t *testing.T,
	output string,
	pattern string,
) int {
	t.Helper()
	value, err := strconv.Atoi(
		parseTextField(t, output, pattern),
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func parseInt64Field(
	t *testing.T,
	output string,
	pattern string,
) int64 {
	t.Helper()
	value, err := strconv.ParseInt(
		parseTextField(t, output, pattern),
		10,
		64,
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}

func parseFloatField(
	t *testing.T,
	output string,
	pattern string,
) float64 {
	t.Helper()
	value, err := strconv.ParseFloat(
		parseTextField(t, output, pattern),
		64,
	)
	if err != nil {
		t.Fatal(err)
	}
	return value
}
