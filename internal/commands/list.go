package commands

import (
	"context"
	"fmt"

	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var cmdList = &cli.Command{
	Name:   "list",
	Action: doList,
}

func init() {
	Commands = append(Commands, cmdList)
}

func doList(c *cli.Context) error {
	store := metadata.GetStore(c.App.Metadata)

	ctx := context.Background()
	l, err := store.List(ctx)
	if err != nil {
		return err
	}
	for _, v := range l {
		fmt.Fprintf(c.App.Writer, "%v\n", v)
	}
	return nil
}
