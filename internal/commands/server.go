package commands

import (
	"path/filepath"

	"github.com/kamichidu/go-netclip/internal"
	"github.com/kamichidu/go-netclip/netclippb/server"
	"github.com/urfave/cli/v2"
)

var cmdServer = &cli.Command{
	Name:   "server",
	Action: doServer,
}

func init() {
	Commands = append(Commands, cmdServer)
}

func doServer(c *cli.Context) error {
	return server.Run(&server.RunConfig{
		Addr:     "127.0.0.1:8000",
		StateDir: filepath.Join(internal.StateDir, "netclip"),
	})
}
