package main

import (
	"context"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/urfave/cli/v3"
)

func main() {
	cmd := cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "log-level",
				Value: "info",
				Usage: "log level for outputs",
			},
			&cli.StringFlag{
				Name:  "config-path",
				Value: getDefaultConfigPath(),
				Usage: "path to configuration file",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "configure",
				Usage:  "Configure chatgpt access",
				Action: ConfigureCLICommand,
			},
			{
				Name:   "test",
				Usage:  "Test configured chatgpt configuration",
				Action: TestCLICommand,
			},
			{
				Name:  "generate",
				Usage: "Generate a new README using a provided codebase",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "target",
						Value: ".",
						Usage: "target directory containing source code for README generation",
					},
				},
				Action: GenerateCLICommand,
			},
		},
	}

	if err := cmd.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}
