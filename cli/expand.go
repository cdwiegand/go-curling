package cli

import (
	"strings"

	curl "github.com/cdwiegand/go-curling/context"
	curlerrors "github.com/cdwiegand/go-curling/errors"
	flag "github.com/spf13/pflag"
)

const expandPrefix = "expand-"

// processVariableArgs walks the arguments in order, as curl does, so that a
// --variable only affects --expand-* options that come after it. Each
// --variable defines a variable as it is reached, and each --expand-<option>
// has {{name}} references in its value expanded before being rewritten to the
// plain --<option> for the flag parser. All other arguments pass through
// untouched; value arguments of other options are skipped so a value that
// looks like "--variable" is never mistaken for an option. On error the
// arguments processed so far are returned with it, so options that came
// earlier (such as -s or --stderr) still take effect, as in curl.
func processVariableArgs(args []string, ctx *curl.CurlContext, flags *flag.FlagSet) ([]string, *curlerrors.CurlError) {
	out := make([]string, 0, len(args))
	for i := 0; i < len(args); i++ {
		arg := args[i]

		if arg == "--" {
			out = append(out, args[i:]...)
			break
		}

		if strings.HasPrefix(arg, "--") {
			name, value, hasValue := strings.Cut(arg[2:], "=")
			expand := strings.HasPrefix(name, expandPrefix)
			if expand {
				name = name[len(expandPrefix):]
			}
			f := flags.Lookup(name)
			if f == nil {
				if expand {
					return out, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "Unknown option: --"+expandPrefix+name)
				}
				out = append(out, arg) // let the flag parser report it
				continue
			}
			if f.NoOptDefVal != "" { // boolean-style option, takes no value
				if expand {
					return out, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "--"+expandPrefix+name+" cannot be used: --"+name+" does not take a value")
				}
				out = append(out, arg)
				continue
			}
			if !hasValue {
				if i+1 >= len(args) {
					if expand {
						return out, curlerrors.NewCurlErrorFromString(curlerrors.ERROR_INVALID_ARGS, "--"+expandPrefix+name+" requires a value")
					}
					out = append(out, arg) // let the flag parser report it
					continue
				}
				i++
				value = args[i]
			}
			if expand {
				expanded, err := ctx.ExpandVariables(value)
				if err != nil {
					return out, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_ARGS, "Unable to expand --"+expandPrefix+name, err)
				}
				value = expanded
			}
			if f.Name == "variable" {
				if err := ctx.SetVariable(value); err != nil {
					return out, curlerrors.NewCurlErrorFromStringAndError(curlerrors.ERROR_INVALID_ARGS, "Invalid --variable", err)
				}
			}
			out = append(out, "--"+f.Name+"="+value)
			continue
		}

		out = append(out, arg)
		if strings.HasPrefix(arg, "-") && len(arg) > 1 {
			// short option cluster such as -sS or -so file: if the last
			// option takes a value and has none attached, the next arg is it
			for j := 1; j < len(arg); j++ {
				f := flags.ShorthandLookup(arg[j : j+1])
				if f == nil {
					break
				}
				if f.NoOptDefVal == "" {
					if j == len(arg)-1 && i+1 < len(args) {
						i++
						out = append(out, args[i])
					}
					break
				}
			}
		}
	}
	return out, nil
}
