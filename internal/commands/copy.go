package commands

import (
	"context"
	"io"

	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var cmdCopy = &cli.Command{
	Name:   "copy",
	Action: doCopy,
}

func init() {
	Commands = append(Commands, cmdCopy)
}

func doCopy(c *cli.Context) error {
	store := metadata.GetStore(c.App.Metadata)
	stdin := metadata.GetStdin(c.App.Metadata)

	data, err := io.ReadAll(stdin)
	if err != nil {
		return err
	}

	ctx := context.Background()
	return store.Copy(ctx, string(data))
}
