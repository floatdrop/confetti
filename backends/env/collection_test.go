package env

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFromEnvironSliceValues(t *testing.T) {
	t.Run("strings", func(t *testing.T) {
		t.Setenv("HOSTS", "first,second,third")
		var cfg struct {
			Hosts []string `env:"HOSTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"first", "second", "third"}, cfg.Hosts)
	})

	t.Run("single_item", func(t *testing.T) {
		t.Setenv("HOSTS", "only")
		var cfg struct {
			Hosts []string `env:"HOSTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"only"}, cfg.Hosts)
	})

	t.Run("ints", func(t *testing.T) {
		t.Setenv("PORTS", "80,443")
		var cfg struct {
			Ports []int `env:"PORTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []int{80, 443}, cfg.Ports)
	})

	t.Run("durations", func(t *testing.T) {
		t.Setenv("BACKOFF", "1s,2s,4s")
		var cfg struct {
			Backoff []time.Duration `env:"BACKOFF"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []time.Duration{time.Second, 2 * time.Second, 4 * time.Second}, cfg.Backoff)
	})

	t.Run("text_unmarshaler_items", func(t *testing.T) {
		t.Setenv("LEVELS", "debug,error")
		var cfg struct {
			Levels []Level `env:"LEVELS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []Level{DebugLevel, ErrorLevel}, cfg.Levels)
	})

	t.Run("pointer_items", func(t *testing.T) {
		t.Setenv("PORTS", "80,443")
		var cfg struct {
			Ports []*int `env:"PORTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		if assert.Len(t, cfg.Ports, 2) {
			assert.Equal(t, 80, *cfg.Ports[0])
			assert.Equal(t, 443, *cfg.Ports[1])
		}
	})

	t.Run("pointer_to_slice", func(t *testing.T) {
		t.Setenv("HOSTS", "first,second")
		var cfg struct {
			Hosts *[]string `env:"HOSTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		if assert.NotNil(t, cfg.Hosts) {
			assert.Equal(t, []string{"first", "second"}, *cfg.Hosts)
		}
	})

	t.Run("named_slice_type", func(t *testing.T) {
		type hosts []string
		t.Setenv("HOSTS", "first,second")
		var cfg struct {
			Hosts hosts `env:"HOSTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, hosts{"first", "second"}, cfg.Hosts)
	})

	t.Run("replaces_current_value", func(t *testing.T) {
		t.Setenv("HOSTS", "loaded")
		cfg := struct {
			Hosts []string `env:"HOSTS"`
		}{Hosts: []string{"first", "second"}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"loaded"}, cfg.Hosts)
	})

	t.Run("empty_value", func(t *testing.T) {
		t.Setenv("HOSTS", "")
		cfg := struct {
			Hosts []string `env:"HOSTS"`
		}{Hosts: []string{"first"}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.NotNil(t, cfg.Hosts)
		assert.Empty(t, cfg.Hosts)
	})

	t.Run("unset_value_is_kept", func(t *testing.T) {
		cfg := struct {
			Hosts []string `env:"HOSTS_UNSET"`
		}{Hosts: []string{"first"}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"first"}, cfg.Hosts)
	})

	t.Run("item_decode_error", func(t *testing.T) {
		t.Setenv("PORTS", "80,http")
		cfg := struct {
			Ports []int `env:"PORTS"`
		}{Ports: []int{1}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.ErrorContains(t, err, `cannot set env variable "PORTS"`)
		assert.ErrorContains(t, err, "cannot decode item 1 of []int")
		assert.Equal(t, []int{1}, cfg.Ports, "target must be kept on error")
	})
}

func TestFromEnvironByteValues(t *testing.T) {
	t.Run("byte_slice_takes_raw_value", func(t *testing.T) {
		t.Setenv("KEY", "a,b,c")
		var cfg struct {
			Key []byte `env:"KEY"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []byte("a,b,c"), cfg.Key)
	})

	t.Run("named_byte_slice", func(t *testing.T) {
		type secret []byte
		t.Setenv("KEY", "value")
		var cfg struct {
			Key secret `env:"KEY"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, secret("value"), cfg.Key)
	})

	t.Run("byte_array", func(t *testing.T) {
		t.Setenv("KEY", "abc")
		var cfg struct {
			Key [4]byte `env:"KEY"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, [4]byte{'a', 'b', 'c', 0}, cfg.Key)
	})

	t.Run("byte_array_overflow", func(t *testing.T) {
		t.Setenv("KEY", "abcde")
		var cfg struct {
			Key [4]byte `env:"KEY"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.ErrorContains(t, err, "cannot decode 5 bytes into [4]uint8")
		assert.Equal(t, [4]byte{}, cfg.Key)
	})

	t.Run("named_byte_items_are_numbers", func(t *testing.T) {
		type code uint8
		t.Setenv("CODES", "1,2")
		var cfg struct {
			Codes []code `env:"CODES"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []code{1, 2}, cfg.Codes)
	})
}

func TestFromEnvironArrayValues(t *testing.T) {
	t.Run("exact_length", func(t *testing.T) {
		t.Setenv("PORTS", "80,443")
		var cfg struct {
			Ports [2]int `env:"PORTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, [2]int{80, 443}, cfg.Ports)
	})

	t.Run("missing_items_are_zeroed", func(t *testing.T) {
		t.Setenv("PORTS", "80")
		cfg := struct {
			Ports [3]int `env:"PORTS"`
		}{Ports: [3]int{1, 2, 3}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, [3]int{80, 0, 0}, cfg.Ports)
	})

	t.Run("too_many_items", func(t *testing.T) {
		t.Setenv("PORTS", "80,443,8080")
		cfg := struct {
			Ports [2]int `env:"PORTS"`
		}{Ports: [2]int{1, 2}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.ErrorContains(t, err, "cannot decode 3 items into [2]int")
		assert.Equal(t, [2]int{1, 2}, cfg.Ports)
	})
}

func TestFromEnvironMapValues(t *testing.T) {
	t.Run("strings", func(t *testing.T) {
		t.Setenv("LABELS", "env=prod,tier=web")
		var cfg struct {
			Labels map[string]string `env:"LABELS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "prod", "tier": "web"}, cfg.Labels)
	})

	t.Run("typed_keys_and_values", func(t *testing.T) {
		t.Setenv("WEIGHTS", "1=0.5,2=1.5")
		var cfg struct {
			Weights map[int]float64 `env:"WEIGHTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, map[int]float64{1: 0.5, 2: 1.5}, cfg.Weights)
	})

	t.Run("value_keeps_extra_pair_separators", func(t *testing.T) {
		t.Setenv("LABELS", "query=a=b")
		var cfg struct {
			Labels map[string]string `env:"LABELS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"query": "a=b"}, cfg.Labels)
	})

	t.Run("replaces_current_value", func(t *testing.T) {
		t.Setenv("LABELS", "env=prod")
		cfg := struct {
			Labels map[string]string `env:"LABELS"`
		}{Labels: map[string]string{"tier": "web"}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "prod"}, cfg.Labels)
	})

	t.Run("empty_value", func(t *testing.T) {
		t.Setenv("LABELS", "")
		cfg := struct {
			Labels map[string]string `env:"LABELS"`
		}{Labels: map[string]string{"tier": "web"}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.NotNil(t, cfg.Labels)
		assert.Empty(t, cfg.Labels)
	})

	t.Run("missing_pair_separator", func(t *testing.T) {
		t.Setenv("LABELS", "env=prod,tier")
		cfg := struct {
			Labels map[string]string `env:"LABELS"`
		}{Labels: map[string]string{"kept": "value"}}

		err := FromEnviron()(t.Context(), &cfg)

		assert.ErrorContains(t, err, `cannot decode "tier" as "=" separated key and value`)
		assert.Equal(t, map[string]string{"kept": "value"}, cfg.Labels)
	})

	t.Run("key_decode_error", func(t *testing.T) {
		t.Setenv("WEIGHTS", "first=1")
		var cfg struct {
			Weights map[int]int `env:"WEIGHTS"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.ErrorContains(t, err, `cannot decode key "first" of map[int]int`)
	})
}

func TestFromEnvironCollectionSeparators(t *testing.T) {
	t.Run("list_separator", func(t *testing.T) {
		t.Setenv("HOSTS", "first;second")
		var cfg struct {
			Hosts []string `env:"HOSTS"`
		}

		err := FromEnviron(ListSeparator(";"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, cfg.Hosts)
	})

	t.Run("pair_separator", func(t *testing.T) {
		t.Setenv("LABELS", "env:prod,tier:web")
		var cfg struct {
			Labels map[string]string `env:"LABELS"`
		}

		err := FromEnviron(PairSeparator(":"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, map[string]string{"env": "prod", "tier": "web"}, cfg.Labels)
	})
}

func TestFromVarCollectionValue(t *testing.T) {
	t.Setenv("HOSTS", "first,second")
	var hosts []string

	err := FromVar("HOSTS")(t.Context(), &hosts)

	assert.NoError(t, err)
	assert.Equal(t, []string{"first", "second"}, hosts)
}

func TestFromEnvironStructCollectionsAreStillTraversed(t *testing.T) {
	type nested struct {
		Value int `env:"VALUE"`
	}

	t.Run("tagged_slice_of_structs", func(t *testing.T) {
		t.Setenv("APP_ITEMS", "ignored")
		t.Setenv("APP_ITEMS_VALUE", "123")
		cfg := struct {
			Items []nested `env:"ITEMS"`
		}{Items: make([]nested, 2)}

		err := FromEnviron(RecursiveKeys, Prefix("APP"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, 123, cfg.Items[0].Value)
		assert.Equal(t, 123, cfg.Items[1].Value)
	})

	t.Run("tagged_map_of_structs", func(t *testing.T) {
		t.Setenv("APP_ITEMS", "ignored")
		t.Setenv("APP_ITEMS_VALUE", "123")
		cfg := struct {
			Items map[string]nested `env:"ITEMS"`
		}{Items: map[string]nested{"first": {}}}

		err := FromEnviron(RecursiveKeys, Prefix("APP"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, 123, cfg.Items["first"].Value)
	})
}

func TestFromEnvironStrict(t *testing.T) {
	t.Run("unsupported_target", func(t *testing.T) {
		t.Setenv("VALUE", "loaded")
		var cfg struct {
			Value any `env:"VALUE"`
		}

		err := FromEnviron(Strict)(t.Context(), &cfg)

		assert.ErrorContains(t, err, `cannot set env variable "VALUE": unsupported target type interface {}`)
	})

	t.Run("unsupported_target_without_strict", func(t *testing.T) {
		t.Setenv("VALUE", "loaded")
		var cfg struct {
			Value any `env:"VALUE"`
		}

		err := FromEnviron()(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Nil(t, cfg.Value)
	})

	t.Run("unset_target_is_not_reported", func(t *testing.T) {
		var cfg struct {
			Value any `env:"VALUE_UNSET"`
		}

		err := FromEnviron(Strict)(t.Context(), &cfg)

		assert.NoError(t, err)
	})

	t.Run("prefix_tags_are_not_reported", func(t *testing.T) {
		type nested struct {
			Value int `env:"VALUE"`
		}
		t.Setenv("APP_NESTED", "ignored")
		t.Setenv("APP_ITEMS", "ignored")
		t.Setenv("APP_NESTED_VALUE", "123")
		var cfg struct {
			Nested nested   `env:"NESTED"`
			Items  []nested `env:"ITEMS"`
		}

		err := FromEnviron(Strict, RecursiveKeys, Prefix("APP"))(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, 123, cfg.Nested.Value)
	})

	t.Run("collections_are_not_reported", func(t *testing.T) {
		t.Setenv("HOSTS", "first,second")
		var cfg struct {
			Hosts []string `env:"HOSTS"`
		}

		err := FromEnviron(Strict)(t.Context(), &cfg)

		assert.NoError(t, err)
		assert.Equal(t, []string{"first", "second"}, cfg.Hosts)
	})
}
