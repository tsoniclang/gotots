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

func requiredCostTool(t *testing.T, name string) string {
	t.Helper()
	path, err := exec.LookPath(name)
	if err != nil {
		t.Fatalf(
			"Stage-2 certification requires %s: %v",
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
	setsidBinary string,
	prlimitBinary string,
	timeoutBinary string,
	timeoutSeconds int,
	command string,
	arguments ...string,
) *exec.Cmd {
	t.Helper()
	commandArguments := []string{
		"--as=3221225472",
		"--",
		timeoutBinary,
		"--signal=TERM",
		"--kill-after=10s",
		fmt.Sprintf("%ds", timeoutSeconds),
		command,
	}
	commandArguments = append(commandArguments, arguments...)
	groupArguments := append(
		[]string{"--wait", prlimitBinary},
		commandArguments...,
	)
	cmd := exec.Command(setsidBinary, groupArguments...)
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
	setsidBinary string,
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
		"--as=3221225472",
		"--",
		timeoutBinary,
		"--signal=TERM",
		"--kill-after=10s",
		fmt.Sprintf("%ds", timeoutSeconds),
		timeBinary,
	}
	commandArguments = append(commandArguments, timedArguments...)
	groupArguments := append(
		[]string{"--wait", prlimitBinary},
		commandArguments...,
	)
	cmd := exec.Command(setsidBinary, groupArguments...)
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
		"GOMEMLIMIT": "448MiB",
		"GOGC":       "20",
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
	for _, key := range []string{
		"GOMEMLIMIT", "GOGC", "GOMAXPROCS", "GOFLAGS",
	} {
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
