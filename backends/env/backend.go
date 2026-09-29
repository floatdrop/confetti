package env

import (
	"context"
	"database/sql"
	"encoding"
	"errors"
	"fmt"
	"os"
	"reflect"
	"time"

	ref "golang.yandex/confetti/internal/reflect"
)

type environOpts struct {
	tag             string
	constructPrefix bool
	prefix          string
	prefixGlue      string
	collection      ref.CollectionOpts
	strict          bool
}

func defaultEnvironOpts() environOpts {
	return environOpts{
		tag:        "env",
		prefixGlue: "_",
		collection: ref.DefaultCollectionOpts(),
	}
}

type loadContext struct {
	prefix string
	opts   environOpts
}

func (c *loadContext) getKey(tag string) string {
	key := tag
	if c.prefix != "" {
		key = c.prefix + c.opts.prefixGlue + key
	}
	return key
}

func (c *loadContext) getPrefix(tag string) string {
	if !c.opts.constructPrefix || tag == "" {
		return c.prefix
	}
	return c.getKey(tag)
}

// FromEnviron loads values from environ to given struct
func FromEnviron(opts ...EnvironOpt) func(context.Context, any) error {
	envOpts := defaultEnvironOpts()
	for _, opt := range opts {
		opt(&envOpts)
	}

	return func(ctx context.Context, target any) error {
		tv := reflect.ValueOf(target)
		return ref.TraverseStruct(
			ref.NewTraverseContext(ctx, loadContext{prefix: envOpts.prefix, opts: envOpts}),
			tv,
			callback,
		)

	}
}

// FromVar loads value from env variable with given key
func FromVar(key string) func(context.Context, any) error {
	return func(_ context.Context, target any) error {
		val, ok := os.LookupEnv(key)
		if !ok {
			return nil
		}

		fv := reflect.ValueOf(target)
		if !fv.IsValid() || fv.Kind() != reflect.Ptr || fv.IsNil() {
			return fmt.Errorf("target for env key '%s' must be a pointer", key)
		}

		_, err := setValue(fv.Elem(), val, defaultEnvironOpts())
		return err
	}
}

func callback(ctx *ref.TraverseContext[loadContext], field ref.Node) error {
	if !field.HasStructField {
		return nil
	}
	if !field.StructField.IsExported() {
		return ref.ErrSkipNested
	}

	tag := field.StructField.Tag.Get(ctx.Data.opts.tag)
	if tag == "-" {
		return ref.ErrSkipNested
	}
	if tag == "" {
		return nil
	}
	key := ctx.Data.getKey(tag)
	ctx.Data.prefix = ctx.Data.getPrefix(tag)

	val, ok := os.LookupEnv(key)
	if !ok {
		// skip on unsetted variable
		return nil
	}

	applied, err := setValue(field.Value, val, ctx.Data.opts)
	if err != nil {
		return fmt.Errorf("cannot set env variable %q: %w", key, err)
	}
	if applied {
		field.Commit()
		return nil
	}
	if ctx.Data.opts.strict && !ref.IsTraversable(field.Value.Type()) {
		return fmt.Errorf("cannot set env variable %q: unsupported target type %s", key, field.Value.Type())
	}
	return nil
}

func recursiveElemWithNew(target reflect.Value) reflect.Value {
	for target.Kind() == reflect.Ptr {
		// set zero value of proper type
		if target.IsNil() {
			target.Set(reflect.New(target.Type().Elem()))
		}
		// unwrap value
		target = target.Elem()
	}
	return target
}

func setValue(refValue reflect.Value, val string, opts environOpts) (bool, error) {
	temporaryType := refValue.Type()
	for temporaryType.Kind() == reflect.Ptr {
		temporaryType = temporaryType.Elem()
	}

	temporary := reflect.New(temporaryType).Elem()
	if current, ok := pointerTerminal(refValue); ok {
		temporary.Set(current)
	}

	applied, err := setTemporaryValue(temporary, val, opts)
	if err != nil || !applied {
		return false, err
	}

	recursiveElemWithNew(refValue).Set(temporary)
	return true, nil
}

func pointerTerminal(refValue reflect.Value) (reflect.Value, bool) {
	for refValue.Kind() == reflect.Ptr {
		if refValue.IsNil() {
			return reflect.Value{}, false
		}
		refValue = refValue.Elem()
	}
	return refValue, true
}

func setTemporaryValue(refValue reflect.Value, val string, opts environOpts) (bool, error) {
	// get pointer to value to check interfaces implementation properly
	valuePtr := refValue.Addr().Interface()

	if vs, ok := valuePtr.(sql.Scanner); ok {
		err := vs.Scan(val)
		if errors.Is(err, ref.ErrSkipNested) {
			return false, nil
		}
		return err == nil, err
	}
	if vs, ok := valuePtr.(encoding.TextUnmarshaler); ok {
		err := vs.UnmarshalText([]byte(val))
		if errors.Is(err, ref.ErrSkipNested) {
			return false, nil
		}
		return err == nil, err
	}
	if vs, ok := valuePtr.(encoding.BinaryUnmarshaler); ok {
		err := vs.UnmarshalBinary([]byte(val))
		if errors.Is(err, ref.ErrSkipNested) {
			return false, nil
		}
		return err == nil, err
	}

	applied, err := ref.SetCollection(refValue, val, opts.collection, elementDecoder(opts))
	if err != nil {
		return false, err
	}
	if applied {
		return true, nil
	}

	if !supportsSetValue(refValue.Type()) {
		return false, nil
	}

	return true, ref.SetValue(refValue, val)
}

// elementDecoder decodes items of a collection the same way as a standalone value.
func elementDecoder(opts environOpts) ref.ElementDecoder {
	return ref.ElementDecoder{
		Supports: supportsDecode,
		Set: func(target reflect.Value, value string) (bool, error) {
			return setValue(target, value, opts)
		},
	}
}

func supportsDecode(target reflect.Type) bool {
	pointer := reflect.PointerTo(target)
	if pointer.Implements(reflect.TypeFor[sql.Scanner]()) ||
		pointer.Implements(reflect.TypeFor[encoding.TextUnmarshaler]()) ||
		pointer.Implements(reflect.TypeFor[encoding.BinaryUnmarshaler]()) {
		return true
	}
	return supportsSetValue(target)
}

func supportsSetValue(target reflect.Type) bool {
	if target == reflect.TypeFor[time.Duration]() {
		return true
	}

	switch target.Kind() {
	case reflect.String,
		reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64, reflect.Int,
		reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uint,
		reflect.Float32, reflect.Float64,
		reflect.Bool,
		reflect.Complex64, reflect.Complex128:
		return true
	default:
		return false
	}
}
