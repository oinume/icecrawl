package cli

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	"github.com/oinume/icecrawl/log"
)

type ExitStatus int

const (
	ExitOK    ExitStatus = 0
	ExitError ExitStatus = 1
)

func (s ExitStatus) Value() int {
	return int(s)
}

func Execute(in io.Reader, out io.Writer, errOut io.Writer) ExitStatus {
	var rootCommand = &cobra.Command{Use: "icecrawl"}
	rootCommand.SetIn(in)
	rootCommand.SetOut(out)
	rootCommand.SetErr(errOut)
	rootCommand.AddCommand(scrapeCommand)
	//rootCommand.AddCommand(healthCheckCommand)
	//rootCommand.AddCommand(mergerCommand)
	//fetcherCommand.AddCommand(fetcherSyutokenMosiCommand)

	//	envVars := config.MustProcess()
	ctx := context.Background()
	//ctx = config.WithContext(ctx, envVars)
	logger := log.New(os.Stdout)
	ctx = log.WithContext(ctx, logger)

	if err := rootCommand.ExecuteContext(ctx); err != nil {
		_, _ = fmt.Fprintln(errOut, err)
		return ExitError
	}
	return ExitOK
}
