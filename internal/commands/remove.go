package commands

import (
	"context"
	"time"

	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var cmdRemove = &cli.Command{
	Name: "remove",
	Flags: []cli.Flag{
		&cli.DurationFlag{
			Name: "expiry",
		},
		&cli.BoolFlag{
			Name: "purge",
		},
	},
	Action: doRemove,
}

func init() {
	Commands = append(Commands, cmdRemove)
}

func doRemove(c *cli.Context) error {
	expiry := c.Duration("expiry")
	purge := c.Bool("purge")
	timestamps := c.Args().Slice()
	store := metadata.GetStore(c.App.Metadata)

	if purge {
		expiry = time.Second
	}
	if expiry != time.Duration(0) && len(timestamps) > 0 {
		return cli.Exit("invalid args, --expiry/--purge/{timestamp} are exclusive.", 1)
	}

	ctx := context.Background()
	if expiry != time.Duration(0) {
		return store.Expire(ctx, time.Now().Add(-expiry))
	}
	l := make([]time.Time, len(timestamps))
	for i := range timestamps {
		t, err := time.Parse(time.RFC3339, timestamps[i])
		if err != nil {
			return err
		}
		l[i] = t
	}
	return store.Remove(ctx, l...)
}
