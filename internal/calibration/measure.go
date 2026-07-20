package calibration

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Measurement joins one manifest fixture to its authored hand port.
// Hand-port bytes are counted after stripping the leading comment
// header so the ratio compares code to code; Go bytes come from the
// manifest span, which includes the Go doc comment the same way the
// port retains in-body source comments.
type Measurement struct {
	FixtureID      string  `json:"fixtureId"`
	GoBytes        int     `json:"goBytes"`
	GeneratedBytes int     `json:"generatedBytes"`
	Verdict        string  `json:"verdict"`
	Status         string  `json:"status"`
	HandPortBytes  int     `json:"handPortBytes,omitempty"`
	HandPortTokens int     `json:"handPortTokens,omitempty"`
	HandPortRatio  float64 `json:"handPortRatio,omitempty"`
	GeneratedRatio float64 `json:"generatedRatio"`
}

// MeasurementReport is the durable joined output; the summary is
// derived from the rows and carries the denominator names required by
// ADR 0011.
type MeasurementReport struct {
	SchemaVersion int           `json:"schemaVersion"`
	Fixtures      []Measurement `json:"fixtures"`
	Summary       struct {
		FixturesTotal          int      `json:"fixturesTotal"`
		FixturesAuthored       int      `json:"fixturesAuthored"`
		FixturesPending        []string `json:"fixturesPending"`
		OrdinaryHandPortMedian float64  `json:"ordinaryHandPortMedianRatio"`
		OrdinaryHandPortDenom  string   `json:"ordinaryHandPortMedianDenominator"`
		AuthoredHandPortBytes  int      `json:"authoredHandPortBytes"`
		AuthoredGoBytes        int      `json:"authoredGoBytes"`
	} `json:"summary"`
}

// stripHeader removes the leading //-comment block and following blank
// lines: the header addresses the reviewer, not the port.
func stripHeader(source []byte) []byte {
	lines := strings.SplitAfter(string(source), "\n")
	start := 0
	for start < len(lines) {
		trimmed := strings.TrimSpace(lines[start])
		if strings.HasPrefix(trimmed, "//") || trimmed == "" {
			start++
			continue
		}
		break
	}
	return []byte(strings.Join(lines[start:], ""))
}

// countTokens is a deterministic lexical count: an identifier/number
// run or a string literal is one token, every other non-space byte is
// one token, comments and whitespace are zero.
func countTokens(source []byte) int {
	tokens := 0
	i := 0
	isWord := func(b byte) bool {
		return b == '_' || b == '$' || b >= '0' && b <= '9' ||
			b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z'
	}
	for i < len(source) {
		b := source[i]
		switch {
		case b == ' ' || b == '\t' || b == '\n' || b == '\r':
			i++
		case b == '/' && i+1 < len(source) && source[i+1] == '/':
			for i < len(source) && source[i] != '\n' {
				i++
			}
		case b == '/' && i+1 < len(source) && source[i+1] == '*':
			i += 2
			for i+1 < len(source) && !(source[i] == '*' && source[i+1] == '/') {
				i++
			}
			i += 2
		case b == '"' || b == '\'' || b == '`':
			quote := b
			i++
			for i < len(source) && source[i] != quote {
				if source[i] == '\\' {
					i++
				}
				i++
			}
			i++
			tokens++
		case isWord(b):
			for i < len(source) && isWord(source[i]) {
				i++
			}
			tokens++
		default:
			i++
			tokens++
		}
	}
	return tokens
}

func round2(v float64) float64 {
	return float64(int(v*100+0.5)) / 100
}

// Measure joins the manifest to the hand-port directory and writes the
// measurement report.
func Measure(manifest *Manifest, handportDir string) (*MeasurementReport, error) {
	report := &MeasurementReport{SchemaVersion: 1}
	var ordinaryRatios []float64
	for _, fixture := range manifest.Fixtures {
		row := Measurement{
			FixtureID:      fixture.FixtureID,
			GoBytes:        fixture.GoBytes,
			GeneratedBytes: fixture.BaselineArtifactBytes,
			Verdict:        fixture.CandidateVerdict,
		}
		if fixture.GoBytes > 0 {
			row.GeneratedRatio = round2(float64(fixture.BaselineArtifactBytes) / float64(fixture.GoBytes))
		}
		portPath := filepath.Join(handportDir, fixture.FixtureID+".ts")
		data, err := os.ReadFile(portPath)
		switch {
		case err == nil:
			code := stripHeader(data)
			row.Status = "authored"
			row.HandPortBytes = len(code)
			row.HandPortTokens = countTokens(code)
			if fixture.GoBytes > 0 {
				row.HandPortRatio = round2(float64(len(code)) / float64(fixture.GoBytes))
			}
			if fixture.CandidateVerdict == "ordinary" {
				ordinaryRatios = append(ordinaryRatios, row.HandPortRatio)
			}
			report.Summary.FixturesAuthored++
			report.Summary.AuthoredHandPortBytes += len(code)
			report.Summary.AuthoredGoBytes += fixture.GoBytes
		case os.IsNotExist(err):
			row.Status = "pending"
			report.Summary.FixturesPending = append(report.Summary.FixturesPending, fixture.FixtureID)
		default:
			return nil, fmt.Errorf("read hand port %s: %w", portPath, err)
		}
		report.Fixtures = append(report.Fixtures, row)
	}
	report.Summary.FixturesTotal = len(manifest.Fixtures)
	report.Summary.OrdinaryHandPortDenom = "authored ordinary-verdict fixtures, hand-port bytes over manifest Go span bytes"
	if len(ordinaryRatios) > 0 {
		sort.Float64s(ordinaryRatios)
		middle := len(ordinaryRatios) / 2
		if len(ordinaryRatios)%2 == 1 {
			report.Summary.OrdinaryHandPortMedian = ordinaryRatios[middle]
		} else {
			report.Summary.OrdinaryHandPortMedian = round2((ordinaryRatios[middle-1] + ordinaryRatios[middle]) / 2)
		}
	}
	return report, nil
}

// WriteMeasurements persists the report beside the manifest.
func WriteMeasurements(report *MeasurementReport, path string) error {
	data, err := json.MarshalIndent(report, "", " ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o644)
}
