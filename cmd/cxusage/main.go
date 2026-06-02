package main

import (
	"errors"
	"flag"
	"fmt"
	"os"

	tea "charm.land/bubbletea/v2"
	"github.com/hashmil/cxusage/internal/cli"
	"github.com/hashmil/cxusage/internal/tui"
)

func main() {
	opts, err := cli.ParseArgs(os.Args[1:])
	if errors.Is(err, flag.ErrHelp) {
		fmt.Print(cli.HelpText("cxusage"))
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "cxusage:", err)
		fmt.Fprintln(os.Stderr)
		fmt.Fprint(os.Stderr, cli.HelpText("cxusage"))
		os.Exit(2)
	}

	if opts.JSON {
		report, err := cli.BuildReport(opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "cxusage:", err)
			os.Exit(1)
		}
		if err := cli.WriteJSON(os.Stdout, report); err != nil {
			fmt.Fprintln(os.Stderr, "cxusage:", err)
			os.Exit(1)
		}
		return
	}

	model := tui.NewModel(tui.Options{
		CodexHome:   opts.CodexHome,
		GroupBy:     opts.GroupBy,
		Timeframe:   opts.Timeframe,
		Theme:       opts.Theme,
		NoColor:     opts.NoColor,
		NoAnimation: opts.NoAnimation,
	})
	if _, err := tea.NewProgram(model).Run(); err != nil {
		fmt.Fprintln(os.Stderr, "cxusage:", err)
		os.Exit(1)
	}
}
