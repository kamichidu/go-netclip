package commands

import (
	"context"
	"io"

	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli"
)

var cmdPaste = cli.Command{
	Name:   "paste",
	Action: doPaste,
}

func init() {
	Commands = append(Commands, cmdPaste)
}

func doPaste(c *cli.Context) error {
	store := metadata.GetStore(c.App.Metadata)

	ctx := context.Background()
	data, err := store.Paste(ctx)
	if err != nil {
		return err
	}
	_, err = io.WriteString(c.App.Writer, data)
	return err
}
