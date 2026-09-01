package implementation

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

type CertificationSource struct {
	sourcePath   string
	sourceDigest string
}

func NewCertificationSource(
	sourcePath string,
	sourceDigest string,
) (CertificationSource, error) {
	decoded, err := hex.DecodeString(sourceDigest)
	if !filepath.IsAbs(sourcePath) || filepath.Clean(sourcePath) != sourcePath ||
		!strings.HasSuffix(sourcePath, ".d.ts") || err != nil || len(decoded) != sha256.Size {
		return CertificationSource{}, certificationError(
			"admit source",
			sourcePath,
			"source evidence is invalid",
		)
	}
	return CertificationSource{
		sourcePath: sourcePath, sourceDigest: sourceDigest,
	}, nil
}

func LoadCertificationSources(paths []string) ([]CertificationSource, error) {
	selected := slices.Clone(paths)
	slices.Sort(selected)
	result := make([]CertificationSource, len(selected))
	for index, path := range selected {
		if index > 0 && selected[index-1] == path {
			return nil, certificationError("select sources", path, "path is duplicated")
		}
		payload, err := os.ReadFile(path)
		if err != nil {
			return nil, certificationError("read source", path, err.Error())
		}
		digest := sha256.Sum256(payload)
		result[index], err = NewCertificationSource(path, hex.EncodeToString(digest[:]))
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

func MergeCertificationSources(
	groups ...[]CertificationSource,
) ([]CertificationSource, error) {
	count := 0
	for _, group := range groups {
		count += len(group)
	}
	result := make([]CertificationSource, 0, count)
	for _, group := range groups {
		result = append(result, group...)
	}
	slices.SortFunc(result, func(left, right CertificationSource) int {
		return strings.Compare(left.sourcePath, right.sourcePath)
	})
	for index, source := range result {
		if !source.Valid() {
			return nil, certificationError(
				"merge sources",
				source.sourcePath,
				"source evidence is invalid",
			)
		}
		if index > 0 && result[index-1].sourcePath == source.sourcePath {
			return nil, certificationError(
				"merge sources",
				source.sourcePath,
				"path is duplicated",
			)
		}
	}
	return result, nil
}

func VerifyCertificationSource(source CertificationSource) ([]byte, error) {
	if !source.Valid() {
		return nil, certificationError(
			"verify source",
			source.sourcePath,
			"source evidence is invalid",
		)
	}
	payload, err := os.ReadFile(source.sourcePath)
	if err != nil {
		return nil, certificationError("read source", source.sourcePath, err.Error())
	}
	digest := sha256.Sum256(payload)
	if hex.EncodeToString(digest[:]) != source.sourceDigest {
		return nil, certificationError("verify source", source.sourcePath, "source digest changed")
	}
	return payload, nil
}

func (s CertificationSource) SourcePath() string   { return s.sourcePath }
func (s CertificationSource) SourceDigest() string { return s.sourceDigest }

func (s CertificationSource) Valid() bool {
	decoded, err := hex.DecodeString(s.sourceDigest)
	return filepath.IsAbs(s.sourcePath) && filepath.Clean(s.sourcePath) == s.sourcePath &&
		strings.HasSuffix(s.sourcePath, ".d.ts") && err == nil && len(decoded) == sha256.Size
}

type CertificationError struct {
	Operation string
	Subject   string
	Reason    string
}

func (e *CertificationError) Error() string {
	if e.Subject == "" {
		return fmt.Sprintf("certify implementation environment %s: %s", e.Operation, e.Reason)
	}
	return fmt.Sprintf(
		"certify implementation environment %s %q: %s",
		e.Operation,
		e.Subject,
		e.Reason,
	)
}

func certificationError(operation string, subject string, reason string) error {
	return &CertificationError{Operation: operation, Subject: subject, Reason: reason}
}
