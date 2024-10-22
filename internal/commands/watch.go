package commands

import (
	"context"
	"fmt"

	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli"
)

var cmdWatch = cli.Command{
	Name:   "watch",
	Action: doWatch,
}

func init() {
	Commands = append(Commands, cmdWatch)
}

func doWatch(c *cli.Context) error {
	store := metadata.GetStore(c.App.Metadata)

	ctx := context.Background()
	ch := store.Watch(ctx)
	for evt := range ch {
		fmt.Fprintf(c.App.Writer, "%v\n", evt)
	}
	return nil
}
