package metadata

import (
	"io"

	"github.com/kamichidu/go-netclip/clipboard"
	"github.com/kamichidu/go-netclip/internal/config"
)

func SetStdin(m map[string]any, v io.Reader) {
	m["stdin"] = v
}

func GetStdin(m map[string]any) io.Reader {
	return m["stdin"].(io.Reader)
}

func SetConfig(m map[string]any, v *config.NetclipConfig) {
	m["config"] = v
}

func GetConfig(m map[string]any) *config.NetclipConfig {
	return m["config"].(*config.NetclipConfig)
}

func SetStore(m map[string]any, v clipboard.Store) {
	m["store"] = v
}

func GetStore(m map[string]any) clipboard.Store {
	return m["store"].(clipboard.Store)
}
