package reflect

import (
	"fmt"
	"reflect"
	"strings"
)

const (
	defaultListSeparator = ","
	defaultPairSeparator = "="
)

// CollectionOpts configures decoding of a collection from a single string value.
type CollectionOpts struct {
	// ListSeparator separates items of a slice, an array or a map.
	ListSeparator string
	// PairSeparator separates a map key from its value.
	PairSeparator string
}

// DefaultCollectionOpts returns separators used when none are configured.
func DefaultCollectionOpts() CollectionOpts {
	return CollectionOpts{
		ListSeparator: defaultListSeparator,
		PairSeparator: defaultPairSeparator,
	}
}

func (o CollectionOpts) listSeparator() string {
	if o.ListSeparator == "" {
		return defaultListSeparator
	}
	return o.ListSeparator
}

func (o CollectionOpts) pairSeparator() string {
	if o.PairSeparator == "" {
		return defaultPairSeparator
	}
	return o.PairSeparator
}

// ElementDecoder decodes collection elements from strings.
type ElementDecoder struct {
	// Supports reports if a value of given type can be decoded from a string.
	Supports func(reflect.Type) bool
	// Set decodes value into target and reports if the value was applied.
	Set func(target reflect.Value, value string) (bool, error)
}

// supports reports if an element of given type can be decoded from a string.
// Pointers are followed, because decoders allocate them on demand.
func (d ElementDecoder) supports(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	if d.Supports(t) {
		return true
	}
	// byte collections are decoded from the raw string, not as a list
	return isByteCollection(t)
}

// SetCollection decodes value into a slice, an array or a map target.
//
// Byte slices and byte arrays take the raw string. Other collections are split
// by the list separator, and map items are split once more into a key and a
// value by the pair separator. The value fully defines the collection: items
// missing from it are not carried over from the current target value.
//
// It reports ok=false and leaves target untouched when target is not a
// collection, or when its elements cannot be decoded from a string. Such
// collections stay available for struct traversal.
func SetCollection(target reflect.Value, value string, opts CollectionOpts, decoder ElementDecoder) (ok bool, err error) {
	switch target.Kind() {
	case reflect.Slice:
		if isByte(target.Type().Elem()) {
			target.Set(reflect.ValueOf([]byte(value)).Convert(target.Type()))
			return true, nil
		}
		return setSlice(target, value, opts, decoder)
	case reflect.Array:
		if isByte(target.Type().Elem()) {
			return setByteArray(target, value)
		}
		return setArray(target, value, opts, decoder)
	case reflect.Map:
		return setMap(target, value, opts, decoder)
	default:
		return false, nil
	}
}

func setSlice(target reflect.Value, value string, opts CollectionOpts, decoder ElementDecoder) (bool, error) {
	if !decoder.supports(target.Type().Elem()) {
		return false, nil
	}

	items := splitList(value, opts.listSeparator())
	result := reflect.MakeSlice(target.Type(), len(items), len(items))
	for i, item := range items {
		if err := setItem(result.Index(i), item, decoder); err != nil {
			return false, fmt.Errorf("cannot decode item %d of %s: %w", i, target.Type(), err)
		}
	}

	target.Set(result)
	return true, nil
}

func setArray(target reflect.Value, value string, opts CollectionOpts, decoder ElementDecoder) (bool, error) {
	if !decoder.supports(target.Type().Elem()) {
		return false, nil
	}

	items := splitList(value, opts.listSeparator())
	if len(items) > target.Len() {
		return false, fmt.Errorf("cannot decode %d items into %s", len(items), target.Type())
	}

	result := reflect.New(target.Type()).Elem()
	for i, item := range items {
		if err := setItem(result.Index(i), item, decoder); err != nil {
			return false, fmt.Errorf("cannot decode item %d of %s: %w", i, target.Type(), err)
		}
	}

	target.Set(result)
	return true, nil
}

func setByteArray(target reflect.Value, value string) (bool, error) {
	if len(value) > target.Len() {
		return false, fmt.Errorf("cannot decode %d bytes into %s", len(value), target.Type())
	}

	result := reflect.New(target.Type()).Elem()
	reflect.Copy(result, reflect.ValueOf([]byte(value)))

	target.Set(result)
	return true, nil
}

func setMap(target reflect.Value, value string, opts CollectionOpts, decoder ElementDecoder) (bool, error) {
	mapType := target.Type()
	// map keys are not allocated on demand, so pointer keys are left to traversal
	if !decoder.Supports(mapType.Key()) || !decoder.supports(mapType.Elem()) {
		return false, nil
	}

	pairs := splitList(value, opts.listSeparator())
	result := reflect.MakeMapWithSize(mapType, len(pairs))
	for _, pair := range pairs {
		rawKey, rawValue, ok := strings.Cut(pair, opts.pairSeparator())
		if !ok {
			return false, fmt.Errorf("cannot decode %q as %q separated key and value of %s", pair, opts.pairSeparator(), mapType)
		}

		key := reflect.New(mapType.Key()).Elem()
		applied, err := decoder.Set(key, rawKey)
		if err != nil {
			return false, fmt.Errorf("cannot decode key %q of %s: %w", rawKey, mapType, err)
		}
		if !applied {
			return false, fmt.Errorf("cannot decode key %q of %s", rawKey, mapType)
		}

		mapValue := reflect.New(mapType.Elem()).Elem()
		if err := setItem(mapValue, rawValue, decoder); err != nil {
			return false, fmt.Errorf("cannot decode value of key %q of %s: %w", rawKey, mapType, err)
		}

		result.SetMapIndex(key, mapValue)
	}

	target.Set(result)
	return true, nil
}

// setItem decodes a single item, keeping the zero value when the decoder opted out.
func setItem(target reflect.Value, value string, decoder ElementDecoder) error {
	_, err := decoder.Set(target, value)
	return err
}

func splitList(value, separator string) []string {
	if value == "" {
		return nil
	}
	return strings.Split(value, separator)
}

func isByte(t reflect.Type) bool {
	return t == reflect.TypeFor[byte]()
}

func isByteCollection(t reflect.Type) bool {
	switch t.Kind() {
	case reflect.Slice, reflect.Array:
		return isByte(t.Elem())
	default:
		return false
	}
}

// IsTraversable reports if a value of given type has nested values to descend into.
func IsTraversable(t reflect.Type) bool {
	for t.Kind() == reflect.Ptr {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.Struct, reflect.Slice, reflect.Array, reflect.Map:
		return true
	default:
		return false
	}
}
