package env

type EnvironOpt func(*environOpts)

// RecursiveKeys is used in FromEnviron function to enable complex keys construction.
// It remembers parent struct key as parser recursively descends deeper and uses it as key prefix for child values.
//
// Example:
//
//	type Config struct {
//		RemoteHost string `env:"APP_REMOTE_HOST"`
//		Credentials Credentials `env:"APP_CREDENTIALS"`
//	}
//
//	type Credentials struct {
//		Username string `env:"USERNAME"`
//		Password string `env:"PASSWORD"`
//	}
//
// In this example, if RecursiveKeys option passed to FromEnviron function,
// struct will be filled using values from 3 environment variables:
// - APP_REMOTE_HOST
// - APP_CREDENTIALS_USERNAME
// - APP_CREDENTIALS_PASSWORD
func RecursiveKeys(opts *environOpts) {
	opts.constructPrefix = true
}

// RecursiveKeyGlue sets custom string to join complex key parts.
// It must be used in conjunction with RecursiveKeys option.
func RecursiveKeyGlue(glue string) EnvironOpt {
	return func(opts *environOpts) {
		opts.prefixGlue = glue
	}
}

// Tag sets struct tag to look key in.
func Tag(tag string) EnvironOpt {
	return func(opts *environOpts) {
		opts.tag = tag
	}
}

// Prefix sets prefix for every env lookup.
func Prefix(p string) EnvironOpt {
	return func(opts *environOpts) {
		opts.prefix = p
	}
}

// ListSeparator sets string that separates items of a slice, an array or a map.
// Default separator is ",".
//
// Example:
//
//	type Config struct {
//		Hosts []string `env:"HOSTS"`
//	}
//
// With HOSTS="first,second" the Hosts field is filled with []string{"first", "second"}.
//
// Separator is not applied to []byte and [N]byte targets: those take the raw value.
func ListSeparator(separator string) EnvironOpt {
	return func(opts *environOpts) {
		opts.collection.ListSeparator = separator
	}
}

// PairSeparator sets string that separates a map key from its value.
// Default separator is "=".
//
// Example:
//
//	type Config struct {
//		Labels map[string]string `env:"LABELS"`
//	}
//
// With LABELS="env=prod,tier=web" the Labels field is filled with
// map[string]string{"env": "prod", "tier": "web"}.
func PairSeparator(separator string) EnvironOpt {
	return func(opts *environOpts) {
		opts.collection.PairSeparator = separator
	}
}

// Strict makes FromEnviron fail when a set environment variable cannot be applied
// to the tagged field, instead of skipping it silently.
//
// Only fields without nested values are reported, so a tag on a struct, a slice or
// a map keeps working as a key prefix for the values inside it.
func Strict(opts *environOpts) {
	opts.strict = true
}
