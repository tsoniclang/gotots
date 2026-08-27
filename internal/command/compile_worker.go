package command

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/tsoniclang/gotots/internal/config"
)

const (
	compileWorkerCommand       = "__gotots_compile_worker_v1"
	compileWorkerDirectoryName = ".gotots-compile-worker"
	compileWorkerSchemaVersion = 1
	compileWorkerLogLimit      = 64 * 1024
)

type compileWorkerDocument struct {
	SchemaVersion   int                    `json:"schemaVersion"`
	WorkerPID       int                    `json:"workerPid"`
	SemanticDigest  string                 `json:"semanticDigest"`
	Files           []compileWorkerFile    `json:"files"`
	RuntimeManifest *compileWorkerArtifact `json:"runtimeManifest"`
	PackageDocument string                 `json:"packageDocument"`
}

type compileWorkerFile struct {
	OutputPath   string `json:"outputPath"`
	ProtocolHash string `json:"protocolHash"`
}

type compileWorkerArtifact struct {
	OutputPath string `json:"outputPath"`
	Payload    string `json:"payload"`
}

func prepareBuildInWorker(
	ctx context.Context,
	project config.Project,
	outputDirectory string,
) (printPlan, string, error) {
	executable, err := os.Executable()
	if err != nil {
		return printPlan{}, "", commandError("locate compilation worker", err.Error())
	}
	workerDirectory := filepath.Join(outputDirectory, compileWorkerDirectoryName)
	if err := os.Mkdir(workerDirectory, 0o700); err != nil {
		return printPlan{}, "", commandError("create compilation worker", err.Error())
	}
	configPath := filepath.Join(workerDirectory, "project.json")
	handoffPath := filepath.Join(workerDirectory, "handoff.json")
	logPath := filepath.Join(workerDirectory, "worker.log")
	projectDocument, err := project.CanonicalJSON()
	if err != nil {
		return printPlan{}, "", err
	}
	if err := writeExclusive(configPath, projectDocument); err != nil {
		return printPlan{}, "", commandError("write compilation worker project", err.Error())
	}
	log, err := os.OpenFile(logPath, os.O_CREATE|os.O_EXCL|os.O_RDWR, 0o600)
	if err != nil {
		return printPlan{}, "", commandError("create compilation worker log", err.Error())
	}
	worker := exec.CommandContext(
		ctx,
		executable,
		compileWorkerCommand,
		configPath,
		outputDirectory,
		handoffPath,
	)
	worker.Stdout = log
	worker.Stderr = log
	worker.Env = os.Environ()
	runErr := worker.Run()
	closeErr := log.Close()
	if runErr != nil || closeErr != nil {
		detail, readErr := readLogTail(logPath, compileWorkerLogLimit)
		if readErr != nil {
			detail = "read worker log: " + readErr.Error()
		}
		return printPlan{}, "", commandError(
			"run compilation worker",
			fmt.Sprintf("%v; close log: %v\n%s", runErr, closeErr, detail),
		)
	}
	document, err := readCompileWorkerDocument(
		handoffPath,
		outputDirectory,
		worker.Process.Pid,
	)
	if err != nil {
		return printPlan{}, "", err
	}
	if err := os.Remove(configPath); err != nil {
		return printPlan{}, "", commandError("remove compilation worker project", err.Error())
	}
	if err := os.Remove(handoffPath); err != nil {
		return printPlan{}, "", commandError("remove compilation worker handoff", err.Error())
	}
	if err := os.Remove(logPath); err != nil {
		return printPlan{}, "", commandError("remove compilation worker log", err.Error())
	}
	if err := os.Remove(workerDirectory); err != nil {
		return printPlan{}, "", commandError("remove compilation worker", err.Error())
	}
	return document.plan, document.semanticDigest, nil
}

func runCompileWorker(ctx context.Context, arguments []string) error {
	if len(arguments) != 3 {
		return commandError("run compilation worker", "project, output, and handoff paths are required")
	}
	configPath, outputDirectory, handoffPath, err := validateCompileWorkerPaths(arguments)
	if err != nil {
		return err
	}
	project, err := config.Load(config.Request{ConfigPath: configPath})
	if err != nil {
		return err
	}
	plan, semanticDigest, err := prepareBuild(ctx, project, outputDirectory)
	if err != nil {
		return err
	}
	payload, err := encodeCompileWorkerDocument(plan, semanticDigest, os.Getpid())
	if err != nil {
		return err
	}
	if err := writeExclusive(handoffPath, payload); err != nil {
		return commandError("write compilation worker handoff", err.Error())
	}
	return nil
}

type decodedCompileWorkerDocument struct {
	plan           printPlan
	semanticDigest string
}

func readCompileWorkerDocument(
	path string,
	outputDirectory string,
	expectedWorkerPID int,
) (decodedCompileWorkerDocument, error) {
	payload, err := os.ReadFile(path)
	if err != nil {
		return decodedCompileWorkerDocument{}, commandError("read compilation worker handoff", err.Error())
	}
	var document compileWorkerDocument
	decoder := json.NewDecoder(bytes.NewReader(payload))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&document); err != nil {
		return decodedCompileWorkerDocument{}, commandError("decode compilation worker handoff", err.Error())
	}
	var extra json.RawMessage
	if err := decoder.Decode(&extra); err != io.EOF {
		if err == nil {
			err = fmt.Errorf("multiple JSON values")
		}
		return decodedCompileWorkerDocument{}, commandError("decode compilation worker handoff", err.Error())
	}
	if document.SchemaVersion != compileWorkerSchemaVersion ||
		expectedWorkerPID <= 0 || expectedWorkerPID == os.Getpid() ||
		document.WorkerPID != expectedWorkerPID ||
		!isSHA256(document.SemanticDigest) || len(document.Files) == 0 {
		return decodedCompileWorkerDocument{}, commandError("decode compilation worker handoff", "identity is invalid")
	}
	protocolDirectory := filepath.Join(outputDirectory, protocolScratchDirectoryName)
	plan := printPlan{
		files:             make([]printPlanFile, len(document.Files)),
		protocolDirectory: protocolDirectory,
	}
	for index, file := range document.Files {
		digest, err := hex.DecodeString(file.ProtocolHash)
		if err != nil || len(digest) != sha256.Size {
			return decodedCompileWorkerDocument{}, commandError("decode compilation worker handoff", "protocol digest is invalid")
		}
		plan.files[index] = printPlanFile{
			outputPath:   file.OutputPath,
			protocolPath: filepath.Join(protocolDirectory, fmt.Sprintf("%06d.ast", index)),
			protocolHash: [sha256.Size]byte(digest),
		}
	}
	plan.packageDocument, err = base64.StdEncoding.DecodeString(document.PackageDocument)
	if err != nil {
		return decodedCompileWorkerDocument{}, commandError("decode compilation worker handoff", "package document is invalid")
	}
	if document.RuntimeManifest != nil {
		payload, decodeErr := base64.StdEncoding.DecodeString(document.RuntimeManifest.Payload)
		if decodeErr != nil {
			return decodedCompileWorkerDocument{}, commandError("decode compilation worker handoff", "runtime manifest is invalid")
		}
		plan.runtimeManifest = printPlanArtifact{
			outputPath: document.RuntimeManifest.OutputPath,
			payload:    payload,
		}
		plan.hasRuntimeManifest = true
	}
	if err := plan.validate(outputDirectory); err != nil {
		return decodedCompileWorkerDocument{}, err
	}
	return decodedCompileWorkerDocument{plan: plan, semanticDigest: document.SemanticDigest}, nil
}

func encodeCompileWorkerDocument(
	plan printPlan,
	semanticDigest string,
	workerPID int,
) ([]byte, error) {
	if workerPID <= 0 || !isSHA256(semanticDigest) {
		return nil, commandError("encode compilation worker handoff", "identity is invalid")
	}
	document := compileWorkerDocument{
		SchemaVersion:   compileWorkerSchemaVersion,
		WorkerPID:       workerPID,
		SemanticDigest:  semanticDigest,
		Files:           make([]compileWorkerFile, len(plan.files)),
		PackageDocument: base64.StdEncoding.EncodeToString(plan.packageDocument),
	}
	for index, file := range plan.files {
		document.Files[index] = compileWorkerFile{
			OutputPath:   file.outputPath,
			ProtocolHash: hex.EncodeToString(file.protocolHash[:]),
		}
	}
	if plan.hasRuntimeManifest {
		document.RuntimeManifest = &compileWorkerArtifact{
			OutputPath: plan.runtimeManifest.outputPath,
			Payload:    base64.StdEncoding.EncodeToString(plan.runtimeManifest.payload),
		}
	}
	payload, err := json.Marshal(document)
	if err != nil {
		return nil, commandError("encode compilation worker handoff", err.Error())
	}
	return append(payload, '\n'), nil
}

func validateCompileWorkerPaths(arguments []string) (string, string, string, error) {
	configPath := filepath.Clean(arguments[0])
	outputDirectory := filepath.Clean(arguments[1])
	handoffPath := filepath.Clean(arguments[2])
	if !filepath.IsAbs(configPath) || !filepath.IsAbs(outputDirectory) || !filepath.IsAbs(handoffPath) {
		return "", "", "", commandError("run compilation worker", "paths must be absolute")
	}
	workerDirectory := filepath.Join(outputDirectory, compileWorkerDirectoryName)
	if configPath != filepath.Join(workerDirectory, "project.json") ||
		handoffPath != filepath.Join(workerDirectory, "handoff.json") {
		return "", "", "", commandError("run compilation worker", "paths escape the output transaction")
	}
	return configPath, outputDirectory, handoffPath, nil
}

func writeExclusive(path string, payload []byte) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	if _, err := file.Write(payload); err != nil {
		_ = file.Close()
		return err
	}
	return file.Close()
}

func readLogTail(path string, limit int64) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil {
		return "", err
	}
	if info.Size() > limit {
		if _, err := file.Seek(info.Size()-limit, io.SeekStart); err != nil {
			return "", err
		}
	}
	payload, err := io.ReadAll(file)
	return string(payload), err
}

func isSHA256(value string) bool {
	decoded, err := hex.DecodeString(value)
	return err == nil && len(decoded) == sha256.Size
}
