package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/dydanz/codeburn-watcher/application"
	"github.com/dydanz/codeburn-watcher/internal/analytics"
)

func main() {
	root := &cobra.Command{
		Use:   "token-monitor",
		Short: "AI coding agent token usage monitor",
	}

	root.AddCommand(
		collectCmd(),
		reportCmd(),
		analyzeCmd(),
		htmlCmd(),
		mergeCmd(),
		pushCmd(),
		initCmd(),
		scheduleCmd(),
		reconcileCmd(),
		fingerprintCmd(),
	)

	if err := root.Execute(); err != nil {
		os.Exit(1)
	}
}

// --- collect ---

func collectCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "collect",
		Short: "Scan agent logs and store new usage events",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			results, err := application.CollectHandler{Adapters: deps.Adapters, Writer: deps.Writer}.
				Handle(cmd.Context(), application.CollectCommand{})
			if err != nil {
				return err
			}
			total := 0
			for _, r := range results {
				total += r.Inserted
			}
			fmt.Printf("Collected %d new events.\n", total)
			return nil
		},
	}
}

// --- report ---

func reportCmd() *cobra.Command {
	var days int
	var asJSON bool
	var outFile string
	c := &cobra.Command{
		Use:   "report",
		Short: "Show token usage report",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.ReportHandler{Deps: deps}.Handle(cmd.Context(), application.ReportQuery{
				Window:  analytics.QueryWindow{Days: days},
				JSON:    asJSON,
				OutFile: outFile,
			})
		},
	}
	c.Flags().IntVarP(&days, "days", "d", 7, "look-back window in days")
	c.Flags().BoolVar(&asJSON, "json", false, "emit JSON export instead of terminal report")
	c.Flags().StringVar(&outFile, "out", "", "write output to file instead of stdout")
	return c
}

// --- analyze ---

func analyzeCmd() *cobra.Command {
	var days int
	var llm bool
	var agent string
	var asJSON bool
	c := &cobra.Command{
		Use:   "analyze",
		Short: "Deep analysis of coding sessions",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.AnalyzeHandler{Deps: deps}.Handle(cmd.Context(), application.AnalyzeQuery{
				Window: analytics.QueryWindow{Days: days},
				LLM:    llm,
				Agent:  agent,
				JSON:   asJSON,
			})
		},
	}
	c.Flags().IntVarP(&days, "days", "d", 7, "look-back window in days")
	c.Flags().BoolVar(&llm, "llm", false, "pipe payload to local LLM agent")
	c.Flags().StringVar(&agent, "agent", "", "override auto-detected agent (claude|gemini|codex)")
	c.Flags().BoolVar(&asJSON, "json", false, "emit raw payload JSON")
	return c
}

// --- html ---

func htmlCmd() *cobra.Command {
	var days int
	var outFile string
	var teamMode bool
	c := &cobra.Command{
		Use:   "html",
		Short: "Generate HTML dashboard",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.HtmlHandler{Deps: deps}.Handle(cmd.Context(), application.HtmlCommand{
				Window:  analytics.QueryWindow{Days: days},
				OutFile: outFile,
				Team:    teamMode,
			})
		},
	}
	c.Flags().IntVarP(&days, "days", "d", 7, "look-back window in days")
	c.Flags().StringVar(&outFile, "out", "report.html", "output HTML file path")
	c.Flags().BoolVar(&teamMode, "team", false, "render team rollup HTML")
	return c
}

// --- merge ---

func mergeCmd() *cobra.Command {
	var verify bool
	var keysFile string
	var asHTML bool
	var outFile string
	var days int
	c := &cobra.Command{
		Use:   "merge <export.json...>",
		Short: "Merge signed exports from team members",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.MergeHandler{Deps: deps}.Handle(cmd.Context(), application.MergeCommand{
				Files:    args,
				Verify:   verify,
				KeysFile: keysFile,
				HTML:     asHTML,
				OutFile:  outFile,
				Days:     days,
			})
		},
	}
	c.Flags().BoolVar(&verify, "verify", false, "verify signatures on all exports")
	c.Flags().StringVar(&keysFile, "keys", "", "path to keyring JSON file")
	c.Flags().BoolVar(&asHTML, "html", false, "render HTML team report")
	c.Flags().StringVar(&outFile, "out", "", "write output to file")
	c.Flags().IntVarP(&days, "days", "d", 7, "look-back window in days for rendering")
	return c
}

// --- push ---

func pushCmd() *cobra.Command {
	var days int
	c := &cobra.Command{
		Use:   "push",
		Short: "Build and push signed export to team server",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.PushHandler{Deps: deps}.Handle(cmd.Context(), application.PushCommand{
				Window: analytics.QueryWindow{Days: days},
			})
		},
	}
	c.Flags().IntVarP(&days, "days", "d", 7, "look-back window in days")
	return c
}

// --- init ---

func initCmd() *cobra.Command {
	var teamURL string
	c := &cobra.Command{
		Use:   "init",
		Short: "Bootstrap identity and install background scheduler",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.InitHandler{Deps: deps}.Handle(cmd.Context(), application.InitCommand{
				TeamConfigURL: teamURL,
			})
		},
	}
	c.Flags().StringVar(&teamURL, "team", "", "URL to fetch team config from")
	return c
}

// --- schedule ---

func scheduleCmd() *cobra.Command {
	var remove bool
	c := &cobra.Command{
		Use:   "schedule",
		Short: "Install or remove the background collection schedule",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.ScheduleHandler{Deps: deps}.Handle(cmd.Context(), application.ScheduleCommand{
				Remove: remove,
			})
		},
	}
	c.Flags().BoolVar(&remove, "remove", false, "uninstall the scheduler entry")
	return c
}

// --- reconcile ---

func reconcileCmd() *cobra.Command {
	var provider string
	var days int
	c := &cobra.Command{
		Use:   "reconcile",
		Short: "Compare local token counts against provider billing API",
		RunE: func(cmd *cobra.Command, _ []string) error {
			if provider == "" {
				return fmt.Errorf("--provider is required (anthropic|openai)")
			}
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.ReconcileHandler{Deps: deps}.Handle(cmd.Context(), application.ReconcileCommand{
				Provider: provider,
				Days:     days,
			})
		},
	}
	c.Flags().StringVar(&provider, "provider", "", "provider to reconcile (anthropic|openai)")
	c.Flags().IntVarP(&days, "days", "d", 7, "look-back window in days")
	return c
}

// --- fingerprint ---

func fingerprintCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "fingerprint",
		Short: "Print local signing identity fingerprint",
		RunE: func(cmd *cobra.Command, _ []string) error {
			deps, cleanup, err := buildDeps()
			if err != nil {
				return err
			}
			defer cleanup()
			return application.FingerprintHandler{Deps: deps}.Handle(context.Background())
		},
	}
}
