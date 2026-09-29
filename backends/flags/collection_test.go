package flags

import (
	"flag"
	"io"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFromArgsCollectionValues(t *testing.T) {
	t.Run("string_slice", func(t *testing.T) {
		reset := swapOsArgs("app", "-hosts", "first,second")
		defer reset()

		var cfg struct {
			Hosts []string `flag:"hosts"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, cfg.Hosts)
	})

	t.Run("duration_slice", func(t *testing.T) {
		reset := swapOsArgs("app", "-backoff", "1s,2s")
		defer reset()

		var cfg struct {
			Backoff []time.Duration `flag:"backoff"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []time.Duration{time.Second, 2 * time.Second}, cfg.Backoff)
	})

	t.Run("byte_slice", func(t *testing.T) {
		reset := swapOsArgs("app", "-key", "a,b")
		defer reset()

		var cfg struct {
			Key []byte `flag:"key"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []byte("a,b"), cfg.Key)
	})

	t.Run("array", func(t *testing.T) {
		reset := swapOsArgs("app", "-ports", "80,443")
		defer reset()

		var cfg struct {
			Ports [2]int `flag:"ports"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, [2]int{80, 443}, cfg.Ports)
	})

	t.Run("map", func(t *testing.T) {
		reset := swapOsArgs("app", "-labels", "env=prod,tier=web")
		defer reset()

		var cfg struct {
			Labels map[string]string `flag:"labels"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "prod", "tier": "web"}, cfg.Labels)
	})

	t.Run("pointer_to_slice", func(t *testing.T) {
		reset := swapOsArgs("app", "-hosts", "first,second")
		defer reset()

		var cfg struct {
			Hosts *[]string `flag:"hosts"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		if assert.NotNil(t, cfg.Hosts) {
			assert.Equal(t, []string{"first", "second"}, *cfg.Hosts)
		}
	})

	t.Run("item_decode_error", func(t *testing.T) {
		reset := swapOsArgs("app", "-ports", "80,http")
		defer reset()

		var cfg struct {
			Ports []int `flag:"ports"`
		}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.ErrorContains(t, err, "cannot decode item 1 of []int")
		assert.Nil(t, cfg.Ports)
	})

	t.Run("not_passed_keeps_value", func(t *testing.T) {
		reset := swapOsArgs("app", "-other", "ignored")
		defer reset()

		cfg := struct {
			Hosts []string `flag:"hosts"`
		}{Hosts: []string{"default"}}
		fset := flag.NewFlagSet("app", flag.ContinueOnError)
		fset.SetOutput(io.Discard)

		err := FromArgs(FlagsSet(fset))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"default"}, cfg.Hosts)
	})
}

func TestFromCollectionValue(t *testing.T) {
	reset := swapOsArgs("app", "-hosts", "first,second")
	defer reset()

	var hosts []string
	fset := flag.NewFlagSet("app", flag.ContinueOnError)
	fset.SetOutput(io.Discard)

	err := From("hosts", FlagSet(fset))(t.Context(), &hosts)

	assert.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, hosts)
}
