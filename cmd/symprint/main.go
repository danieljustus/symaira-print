// Command symprint turns Markdown (+ a frontmatter contract) into beautiful PDFs
// via named use-case profiles (brief, behoerde, report, rechnung), so AI agents,
// CLIs and MCP clients get consistent output without the pandoc/LaTeX iteration
// pain. The typesetting engine (Typst) is reached over PATH, never linked, so
// symprint stays a single CGO-free binary.
package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"os"

	"github.com/spf13/cobra"

	"github.com/danieljustus/symaira-corekit/exitcodes"
	"github.com/danieljustus/symaira-corekit/logkit"
	"github.com/danieljustus/symaira-print/internal/config"
	"github.com/danieljustus/symaira-print/internal/press"
)

var version = "0.1.0-dev"

// jsonOut is bound to the global --json flag.
var jsonOut bool

func main() {
	slog.SetDefault(logkit.NewFromEnv("symprint"))
	if err := newRootCmd().Execute(); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", exitcodes.FormatCLIError(err))
		os.Exit(int(exitcodes.ExitCodeFromError(err)))
	}
}

func newRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:           "symprint",
		Short:         "Markdown → beautiful PDF via named use-case profiles",
		Version:       version,
		SilenceUsage:  true,
		SilenceErrors: true,
		Long: `symprint renders Markdown (+ a YAML frontmatter contract) into polished PDFs
using named profiles that fix every visual decision for a use case:

  brief      DIN 5008 letter (Form B)
  behoerde   DIN 5008 authority letter (Form A) + PDF/A-2a + PDF/UA-1
  report     cover page, table of contents, headers/footers, page numbers
  rechnung   data-driven German invoice (scaffold)

The engine is Typst, reached over PATH (run 'symprint doctor' to check). The
contract is strict: unknown frontmatter keys are rejected, so what you write is
what you get. Designed to be driven by humans, CLIs, and AI agents (MCP).`,
	}

	root.PersistentFlags().BoolVar(&jsonOut, "json", false, "emit machine-readable JSON")

	root.AddCommand(
		newRenderCmd(),
		newProfilesCmd(),
		newValidateCmd(),
		newDoctorCmd(),
		newConfigCmd(),
		newMCPCmd(),
		newVersionCmd(),
	)
	return root
}

// engineFromConfig maps the loaded config to the engine options press needs.
func engineFromConfig(cfg *config.Config) press.EngineConfig {
	return press.EngineConfig{
		TypstBin:          cfg.Engine.Typst,
		FontPaths:         cfg.Engine.FontPaths,
		IgnoreSystemFonts: cfg.Engine.IgnoreSystemFonts,
		Timeout:           cfg.Engine.Timeout(),
	}
}

// mapPressError translates a typed press error into a corekit CLIError with the
// right exit code, error kind, and an actionable hint — never a raw engine dump.
func mapPressError(err error) error {
	var ce *press.ContractError
	if errors.As(err, &ce) {
		return exitcodes.Wrap(err, exitcodes.ExitData, exitcodes.KindValidation, "document contract")
	}
	var re *press.RenderError
	if errors.As(err, &re) {
		code, kind := exitcodes.ExitGeneric, exitcodes.KindInternal
		switch re.Stage {
		case "engine":
			code, kind = exitcodes.ExitGeneric, exitcodes.KindUnavailable
		case "contract":
			code, kind = exitcodes.ExitData, exitcodes.KindValidation
		case "compile":
			code, kind = exitcodes.ExitData, exitcodes.KindValidation
		case "write":
			code, kind = exitcodes.ExitGeneric, exitcodes.KindInternal
		}
		cliErr := exitcodes.Wrap(err, code, kind, "render")
		cliErr.Hint = re.Hint
		return cliErr
	}
	return exitcodes.Wrap(err, exitcodes.ExitGeneric, exitcodes.KindInternal, "symprint")
}

func printJSON(v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	fmt.Println(string(b))
	return nil
}
