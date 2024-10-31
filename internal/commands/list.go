package commands

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"text/template"
	"time"

	"github.com/kamichidu/go-netclip/clipboard"
	"github.com/kamichidu/go-netclip/internal/metadata"
	"github.com/urfave/cli/v2"
)

var (
	//go:embed templates/list.default.txt
	listDefaultTemplateString string

	listDefaultTemplate *template.Template
)

var cmdList = &cli.Command{
	Name:   "list",
	Action: doList,
}

func init() {
	Commands = append(Commands, cmdList)

	listDefaultTemplate = template.New("list.default")
	listDefaultTemplate = listDefaultTemplate.Funcs(map[string]any{
		"datetime": func(timestamp int64) string {
			t := time.Unix(timestamp, 0)
			return t.Format(time.RFC3339)
		},
		"shorten": clipboard.Shorten,
		"nonl": func(s string) string {
			s = strings.ReplaceAll(s, "\r", "\\r")
			s = strings.ReplaceAll(s, "\n", "\\n")
			return s
		},
	})
	listDefaultTemplate = template.Must(listDefaultTemplate.Parse(listDefaultTemplateString))
}

func doList(c *cli.Context) error {
	store := metadata.GetStore(c.App.Metadata)

	ctx := context.Background()
	l, err := store.List(ctx)
	if err != nil {
		return err
	}
	for _, v := range l {
		if err := listDefaultTemplate.Execute(c.App.Writer, v); err != nil {
			fmt.Fprintf(c.App.ErrWriter, "error: %v\n", err)
		}
	}
	return nil
}
