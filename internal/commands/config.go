package commands

import (
	"errors"
	"fmt"

	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var cmdConfig = &cli.Command{
	Name: "config",
	Subcommands: cli.Commands{
		&cli.Command{
			Name:   "list",
			Action: configList,
		},
		&cli.Command{
			Name:   "get",
			Action: configGet,
		},
		&cli.Command{
			Name:   "set",
			Action: configSet,
		},
	},
}

func init() {
	Commands = append(Commands, cmdConfig)
}

func configList(c *cli.Context) error {
	cfg := metadata.GetConfig(c.App.Metadata)
	for _, k := range cfg.Keys() {
		v := cfg.Get(k)
		fmt.Fprintf(c.App.Writer, "%s=%v\n", k, v)
	}
	return nil
}

func configGet(c *cli.Context) error {
	cfg := metadata.GetConfig(c.App.Metadata)
	key := c.Args().Get(0)
	if !cfg.ValidKey(key) {
		return errors.New("no such key: " + key)
	}
	value := cfg.Get(key)
	fmt.Fprintf(c.App.Writer, "%v\n", value)
	return nil
}

func configSet(c *cli.Context) error {
	cfg := metadata.GetConfig(c.App.Metadata)
	key := c.Args().Get(0)
	value := c.Args().Get(1)
	if !cfg.ValidKey(key) {
		return errors.New("no such key: " + key)
	}
	cfg.Set(key, value)
	return cfg.Commit()
}
