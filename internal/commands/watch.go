package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"strings"

	"github.com/kamichidu/go-netclip/clipboard"
	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var cmdWatch = &cli.Command{
	Name: "watch",
	Flags: []cli.Flag{
		&cli.BoolFlag{
			Name: "stream",
		},
		&cli.StringFlag{
			Name: "exec",
		},
	},
	Action: doWatch,
}

func init() {
	Commands = append(Commands, cmdWatch)
}

func doWatch(c *cli.Context) error {
	stream := c.Bool("stream")
	execCmd := c.String("exec")
	store := metadata.GetStore(c.App.Metadata)

	var write func(clipboard.Event) error
	if stream {
		je := json.NewEncoder(c.App.Writer)
		write = func(evt clipboard.Event) error {
			return je.Encode(evt)
		}
	} else {
		write = func(evt clipboard.Event) error {
			_, err := fmt.Fprintf(c.App.Writer, "%v\n", evt)
			return err
		}
	}

	ctx := context.Background()
	ch := store.Watch(ctx)
	for evt := range ch {
		write(evt)
		if execCmd == "" {
			continue
		}
		cmd := exec.CommandContext(ctx, execCmd)
		cmd.Stdin = strings.NewReader(evt.Value)
		cmd.Stdout = io.Discard
		cmd.Stderr = c.App.ErrWriter
		if err := cmd.Run(); err != nil {
			fmt.Fprintf(c.App.ErrWriter, "error: %s: %v\n", execCmd, err)
			continue
		}
	}
	return nil
}
