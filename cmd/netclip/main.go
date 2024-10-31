package main

import (
	_ "embed"
	"io"
	"log"
	"os"
	"strings"

	"github.com/comail/colog"
	"github.com/kamichidu/go-netclip/clipboard"
	_ "github.com/kamichidu/go-netclip/clipboard/driver/firestore"
	_ "github.com/kamichidu/go-netclip/clipboard/driver/netclipserver"
	"github.com/kamichidu/go-netclip/config"
	"github.com/kamichidu/go-netclip/internal"
	"github.com/kamichidu/go-netclip/internal/commands"
	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var (
	//go:embed usage.txt
	usageString string
)

func init() {
	colog.Register()

	config.Register("driver", config.NewSpec("netclip.server", config.TypeString))
}

func run(stdin io.Reader, stdout, stderr io.Writer, args []string) int {
	log.SetOutput(stderr)

	app := cli.NewApp()
	app.Name = "netclip"
	app.Usage = "Yet another network clipboard sharing tool"
	app.UsageText = strings.TrimSpace(usageString)
	app.Writer = stdout
	app.ErrWriter = stderr
	app.Commands = commands.Commands
	app.Flags = []cli.Flag{
		&cli.StringFlag{
			Name:  "config, c",
			Usage: "configuration file `path`",
			Value: internal.DefaultConfigFile,
		},
	}
	app.Before = func(c *cli.Context) error {
		metadata.SetStdin(c.App.Metadata, stdin)

		filename := c.String("config")
		cfg, err := config.NewNetclipConfigFromFile(filename)
		if err != nil {
			return err
		}
		metadata.SetConfig(c.App.Metadata, cfg)

		driverName, _ := cfg.Get("driver").(string)
		store, err := clipboard.NewStore(driverName, cfg)
		if err != nil {
			return err
		}
		metadata.SetStore(c.App.Metadata, store)
		return nil
	}
	if err := app.Run(args); err != nil {
		log.Printf("error: %v", err)
		return 1
	}
	return 0
}

func main() {
	os.Exit(run(os.Stdin, os.Stdout, os.Stderr, os.Args))
}
