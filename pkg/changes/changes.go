package changes

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	generationaccess "github.com/speakeasy-api/generation-context/access"
	"github.com/speakeasy-api/openapi-generation/v2/internal/analytics"
	"github.com/speakeasy-api/openapi-generation/v2/internal/ast"
	"github.com/speakeasy-api/openapi-generation/v2/internal/document"
	"github.com/speakeasy-api/openapi-generation/v2/internal/generate"
	"github.com/speakeasy-api/openapi-generation/v2/internal/subsystem"
	pkggenerate "github.com/speakeasy-api/openapi-generation/v2/pkg/generate"
	"github.com/speakeasy-api/openapi-generation/v2/pkg/logging"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Exposing GenerateOptions
type GenerateOptions = generate.GenerateOptions

type SpecComparison struct {
	OldSpecPath string
	NewSpecPath string
	OutputDir   string
	Lang        string
	Verbose     bool
	Logger      logging.Logger
}

// Changes compares two GenerateOptions and returns the differences
func Changes(context context.Context, oldConfig, newConfig GenerateOptions) (SDKDiff, error) {
	astOld, subsystemOld, errForOld := resolveAST(context, oldConfig)
	if errForOld != nil {
		return SDKDiff{}, fmt.Errorf("failed to resolve ast. error while resolving ast for old config: %s", errForOld.Error())
	}
	astNew, subsystemNew, errForNew := resolveAST(context, newConfig)
	if errForNew != nil {
		return SDKDiff{}, fmt.Errorf("failed to resolve ast. error while resolving ast for new config: %s", errForNew.Error())
	}

	// Get diff
	return DiffASTs(DiffOptions{
		OldAST:       astOld,
		NewAST:       astNew,
		OldSubsystem: subsystemOld,
		NewSubsystem: subsystemNew,
	}), nil
}

func CreateConfigsFromSpecBytes(oldSpec, newSpec []byte, options GenerateOptions) (oldConfig, newConfig GenerateOptions) {
	// Create temp directory for test files
	tmpDir := os.TempDir()

	oldSpecPath := filepath.Join(tmpDir, "old_spec.yaml")
	newSpecPath := filepath.Join(tmpDir, "new_spec.yaml")

	// Write specs to files
	if err := os.WriteFile(oldSpecPath, oldSpec, 0o644); err != nil {
		options.Logger.Error("Failed to write old spec", zap.Error(err))
	}
	if err := os.WriteFile(newSpecPath, newSpec, 0o644); err != nil {
		options.Logger.Error("Failed to write new spec", zap.Error(err))
	}

	oldConfig = GenerateOptions{
		SchemaPath: oldSpecPath,
		OutDir:     filepath.Join(tmpDir, "old"),
		Lang:       options.Lang,
		Verbose:    options.Verbose,
		Logger:     options.Logger,
	}
	newConfig = GenerateOptions{
		SchemaPath: newSpecPath,
		OutDir:     filepath.Join(tmpDir, "new"),
		Lang:       options.Lang,
		Verbose:    options.Verbose,
		Logger:     options.Logger,
	}

	return oldConfig, newConfig
}

func CreateConfigsFromSpecPaths(config SpecComparison) (oldConfig, newConfig GenerateOptions) {
	oldConfig = GenerateOptions{
		SchemaPath: config.OldSpecPath,
		OutDir:     config.OutputDir,
		Lang:       config.Lang,
		Verbose:    config.Verbose,
		Logger:     config.Logger,
	}
	newConfig = GenerateOptions{
		SchemaPath: config.NewSpecPath,
		OutDir:     config.OutputDir,
		Lang:       config.Lang,
		Verbose:    config.Verbose,
		Logger:     config.Logger,
	}

	return oldConfig, newConfig
}

// resolveAST resolves the AST from GenerateOptions using pkg/generate
func resolveAST(ctx context.Context, config GenerateOptions) (*ast.AST, *subsystem.Subsystem, error) {
	// Ensure we have a logger
	if config.Logger == nil {
		config.Logger = logging.NewLogger(zapcore.DebugLevel)
	}
	// The generator skips operations that use feature-gated constructs, such as
	// server-sent event responses, when no generation access state is present.
	if _, ok := generationaccess.StateFromContext(ctx); !ok {
		ctx = generationaccess.WithDirect(ctx)
	}
	// Check if schema file exists
	if _, err := os.Stat(config.SchemaPath); err != nil {
		// If file doesn't exist, return nil AST and nil Subsystem
		return nil, nil, fmt.Errorf("schema file does not exist. schemaPath: %s. error: %s", config.SchemaPath, err.Error())
	}

	// Read the schema file
	schemaBytes, err := os.ReadFile(config.SchemaPath)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to read schema path. schemaPath: %s. error: %s", config.SchemaPath, err.Error())
	}

	// Create a generator with minimal configuration
	gen, err := pkggenerate.New(
		pkggenerate.WithDontWrite(), // Don't write files, just generate AST
		pkggenerate.WithLogger(config.Logger),
		pkggenerate.WithTracer(config.Tracer),
		pkggenerate.WithVerboseOutput(config.Verbose),
	)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to create generator. error: %s", err.Error())
	}

	// Initialize with target language
	err = gen.Init(ctx, config.Lang, config.OutDir)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to initialize generator. error: %s", err.Error())
	}

	// Create DocumentInfo
	docInfo := &document.DocumentInfo{
		Schema:     schemaBytes,
		SchemaPath: config.SchemaPath,
		IsRemote:   false,
	}

	// Load and validate document
	ad := &analytics.Data{}

	if _, err := gen.LoadAndValidateDoc(ctx, docInfo, ad, "", true); err != nil {
		return nil, nil, fmt.Errorf("failed to load and validate document. error: %s", err.Error())
	}

	// Generate AST
	astTree, err := gen.GenerateAST(ctx, docInfo, ad)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate AST. error: %s", err.Error())
	}

	// Return both AST and Subsystem
	return astTree, gen.GetSubsystem(), nil
}
