package reflect

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/assert"
)

// stringDecoder decodes strings and ints only, like a backend would.
func stringDecoder() ElementDecoder {
	return ElementDecoder{
		Supports: func(t reflect.Type) bool {
			switch t.Kind() {
			case reflect.String, reflect.Int:
				return true
			default:
				return false
			}
		},
		Set: func(target reflect.Value, value string) (bool, error) {
			for target.Kind() == reflect.Ptr {
				if target.IsNil() {
					target.Set(reflect.New(target.Type().Elem()))
				}
				target = target.Elem()
			}
			if ok, err := SetCollection(target, value, DefaultCollectionOpts(), stringDecoder()); ok || err != nil {
				return ok, err
			}
			switch target.Kind() {
			case reflect.String, reflect.Int:
				return true, SetValue(target, value)
			default:
				return false, nil
			}
		},
	}
}

func setCollection[T any](t *testing.T, target *T, value string, opts CollectionOpts) (bool, error) {
	t.Helper()
	return SetCollection(reflect.ValueOf(target).Elem(), value, opts, stringDecoder())
}

func TestSetCollectionUnsupportedTargets(t *testing.T) {
	t.Run("not_a_collection", func(t *testing.T) {
		target := "kept"
		ok, err := setCollection(t, &target, "loaded", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "kept", target)
	})

	t.Run("unsupported_items", func(t *testing.T) {
		target := []struct{ Value string }{{Value: "kept"}}
		ok, err := setCollection(t, &target, "loaded", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "kept", target[0].Value)
	})

	t.Run("unsupported_map_values", func(t *testing.T) {
		target := map[string]struct{ Value string }{}
		ok, err := setCollection(t, &target, "key=value", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.False(t, ok)
	})

	t.Run("pointer_map_keys", func(t *testing.T) {
		target := map[*string]string{}
		ok, err := setCollection(t, &target, "key=value", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Empty(t, target)
	})
}

func TestSetCollectionArrays(t *testing.T) {
	t.Run("items_are_zeroed_before_decoding", func(t *testing.T) {
		target := [3]string{"first", "second", "third"}
		ok, err := setCollection(t, &target, "loaded", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, [3]string{"loaded", "", ""}, target)
	})

	t.Run("too_many_items", func(t *testing.T) {
		target := [1]string{"kept"}
		ok, err := setCollection(t, &target, "first,second", DefaultCollectionOpts())

		assert.ErrorContains(t, err, "cannot decode 2 items into [1]string")
		assert.False(t, ok)
		assert.Equal(t, [1]string{"kept"}, target)
	})

	t.Run("unsupported_items", func(t *testing.T) {
		target := [1]struct{ Value string }{{Value: "kept"}}
		ok, err := setCollection(t, &target, "loaded", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.False(t, ok)
		assert.Equal(t, "kept", target[0].Value)
	})

	t.Run("byte_array_takes_raw_value", func(t *testing.T) {
		target := [4]byte{'k', 'e', 'p', 't'}
		ok, err := setCollection(t, &target, "ab", DefaultCollectionOpts())

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, [4]byte{'a', 'b', 0, 0}, target)
	})

	t.Run("byte_array_overflow", func(t *testing.T) {
		target := [1]byte{'k'}
		ok, err := setCollection(t, &target, "ab", DefaultCollectionOpts())

		assert.ErrorContains(t, err, "cannot decode 2 bytes into [1]uint8")
		assert.False(t, ok)
		assert.Equal(t, [1]byte{'k'}, target)
	})
}

func TestSetCollectionNestedByteSlices(t *testing.T) {
	var target [][]byte
	ok, err := setCollection(t, &target, "first,second", DefaultCollectionOpts())

	assert.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, [][]byte{[]byte("first"), []byte("second")}, target)
}

func TestSetCollectionNestedListsAreNotSupported(t *testing.T) {
	target := [][]string{{"kept"}}
	ok, err := setCollection(t, &target, "first,second", DefaultCollectionOpts())

	assert.NoError(t, err)
	assert.False(t, ok)
	assert.Equal(t, [][]string{{"kept"}}, target)
}

func TestSetCollectionEmptySeparatorsFallBackToDefaults(t *testing.T) {
	t.Run("list", func(t *testing.T) {
		var target []string
		ok, err := setCollection(t, &target, "first,second", CollectionOpts{})

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, []string{"first", "second"}, target)
	})

	t.Run("pair", func(t *testing.T) {
		var target map[string]string
		ok, err := setCollection(t, &target, "key=value", CollectionOpts{})

		assert.NoError(t, err)
		assert.True(t, ok)
		assert.Equal(t, map[string]string{"key": "value"}, target)
	})
}

func TestIsTraversable(t *testing.T) {
	assert.True(t, IsTraversable(reflect.TypeFor[struct{}]()))
	assert.True(t, IsTraversable(reflect.TypeFor[*[]string]()))
	assert.True(t, IsTraversable(reflect.TypeFor[map[string]string]()))
	assert.True(t, IsTraversable(reflect.TypeFor[[2]int]()))
	assert.False(t, IsTraversable(reflect.TypeFor[string]()))
	assert.False(t, IsTraversable(reflect.TypeFor[any]()))
	assert.False(t, IsTraversable(reflect.TypeFor[chan int]()))
}
