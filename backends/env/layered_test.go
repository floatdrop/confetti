package env_test

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"golang.yandex/confetti"
	"golang.yandex/confetti/backends/env"
	"golang.yandex/confetti/backends/json"
)

// TestEnvironOverridesJSONCollections checks that collection values loaded from a
// JSON document can be overridden by environment variables.
func TestEnvironOverridesJSONCollections(t *testing.T) {
	type config struct {
		Hosts  []string          `json:"hosts" env:"HOSTS"`
		Labels map[string]string `json:"labels" env:"LABELS"`
		Port   int               `json:"port" env:"PORT"`
	}

	document := `{"hosts": ["file"], "labels": {"source": "file"}, "port": 80}`

	t.Run("environ_wins", func(t *testing.T) {
		t.Setenv("HOSTS", "first,second")
		t.Setenv("LABELS", "source=environ")
		t.Setenv("PORT", "443")

		var cfg config
		err := confetti.NewLoader().Load(t.Context(),
			confetti.To(&cfg, json.FromString(document), env.FromEnviron()),
		)

		assert.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, cfg.Hosts)
		assert.Equal(t, map[string]string{"source": "environ"}, cfg.Labels)
		assert.Equal(t, 443, cfg.Port)
	})

	t.Run("json_is_kept_without_environ", func(t *testing.T) {
		var cfg config
		err := confetti.NewLoader().Load(t.Context(),
			confetti.To(&cfg, json.FromString(document), env.FromEnviron()),
		)

		assert.NoError(t, err)
		assert.Equal(t, []string{"file"}, cfg.Hosts)
		assert.Equal(t, map[string]string{"source": "file"}, cfg.Labels)
		assert.Equal(t, 80, cfg.Port)
	})
}
