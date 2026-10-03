package common

import (
	"fmt"
	"io"
	"strconv"
	"strings"
)

// Flag is one option of a command. A command's flags are one table: it
// parses the arguments and prints the help, so the help can't miss a flag
// the command takes.
type Flag struct {
	// Names: the short name first ("-c", "--config").
	Names []string
	// Arg names the value in the help ("FILE"); "" is a boolean flag.
	Arg string
	// Usage is the help text; "\n" starts another line.
	Usage string
	set   func(string) error
}

// String is a flag setting *p to its value.
func String(p *string, arg, usage string, names ...string) Flag {
	return Func(arg, usage, func(v string) error { *p = v; return nil }, names...)
}

// Strings is a repeatable flag appending its values to *p.
func Strings(p *[]string, arg, usage string, names ...string) Flag {
	return Func(arg, usage, func(v string) error { *p = append(*p, v); return nil }, names...)
}

// Bool is a boolean flag: "--x" sets true, "--x=false" false.
func Bool(p *bool, usage string, names ...string) Flag {
	return BoolFunc(usage, func(v bool) { *p = v }, names...)
}

// Func is a flag with a value handled by set.
func Func(arg, usage string, set func(string) error, names ...string) Flag {
	return Flag{Names: names, Arg: arg, Usage: usage, set: set}
}

// BoolFunc is a boolean flag handled by set.
func BoolFunc(usage string, set func(bool), names ...string) Flag {
	return Flag{Names: names, Usage: usage, set: func(v string) error {
		b, err := strconv.ParseBool(v)
		if err != nil {
			return err
		}
		set(b)
		return nil
	}}
}

// TakesValue tells whether the flag needs a value.
func (f Flag) TakesValue() bool { return f.Arg != "" }

// Lookup finds the flag named name ("--config"), nil if none.
func Lookup(flags []Flag, name string) *Flag {
	for i := range flags {
		for _, n := range flags[i].Names {
			if n == name {
				return &flags[i]
			}
		}
	}
	return nil
}

// ParseFlags sets flags from args and returns the other arguments in
// order. A value goes after "=" or as the next argument (not one starting
// with "--"); a boolean takes "=true"/"=false" only. command names the
// command in errors ("feedback draft").
//
// Пример: ["--dry-run", "a", "-c", "x.yaml", "--add-relations=false"] ->
// dry-run=true, config=x.yaml, add-relations=false, returns ["a"].
func ParseFlags(args []string, flags []Flag, command string) ([]string, error) {
	var positional []string
	for i := 0; i < len(args); i++ {
		arg := args[i]
		if !strings.HasPrefix(arg, "-") || arg == "-" {
			positional = append(positional, arg)
			continue
		}
		name, value, hasValue := strings.Cut(arg, "=")
		flag := Lookup(flags, name)
		if flag == nil {
			return nil, fmt.Errorf("unknown flag %q for %s: see aism %s --help", name, command, command)
		}
		if !flag.TakesValue() {
			if !hasValue {
				value = "true"
			}
			if err := flag.set(value); err != nil {
				return nil, fmt.Errorf("%s requires a boolean", name)
			}
			continue
		}
		if !hasValue {
			if i+1 == len(args) || strings.HasPrefix(args[i+1], "--") {
				return nil, fmt.Errorf("%s requires a value", name)
			}
			i++
			value = args[i]
		}
		if value == "" {
			return nil, fmt.Errorf("%s requires a value", name)
		}
		if err := flag.set(value); err != nil {
			return nil, fmt.Errorf("%s: %w", name, err)
		}
	}
	return positional, nil
}

// Help is a command's help: usage lines, a description, the actions or
// commands it dispatches to, its flags, then the global ones.
type Help struct {
	Usage       []string
	Description string
	// Entries: name and summary of each action (or command).
	EntriesTitle string
	Entries      [][2]string
	Flags        []Flag
	Global       []Flag
	Footer       string
}

// String renders the help.
//
// Пример:
//
//	Usage: aism validate [options]
//
//	Check the configuration and the skills without writing anything.
//
//	Options:
//	  -c, --config FILE  YAML or JSON config (default: ai-skills.yaml)
//	...
func (h Help) String() string {
	var b strings.Builder
	for i, line := range h.Usage {
		prefix := "Usage: "
		if i > 0 {
			prefix = "       "
		}
		b.WriteString(prefix + line + "\n")
	}
	if h.Description != "" {
		b.WriteString("\n" + h.Description + "\n")
	}
	if len(h.Entries) > 0 {
		b.WriteString("\n" + h.EntriesTitle + ":\n")
		width := 0
		for _, e := range h.Entries {
			width = max(width, len(e[0]))
		}
		for _, e := range h.Entries {
			writeColumns(&b, e[0], e[1], width)
		}
	}
	if len(h.Flags) > 0 {
		b.WriteString("\nOptions:\n")
		writeFlags(&b, h.Flags)
	}
	if len(h.Global) > 0 {
		b.WriteString("\nGlobal options:\n")
		writeFlags(&b, h.Global)
	}
	if h.Footer != "" {
		b.WriteString("\n" + h.Footer + "\n")
	}
	return b.String()
}

func writeFlags(w io.Writer, flags []Flag) {
	heads := make([]string, len(flags))
	width := 0
	for i, f := range flags {
		head := strings.Join(f.Names, ", ")
		if strings.HasPrefix(head, "--") {
			// Lined up with the long name of "-c, --config".
			head = "    " + head
		}
		if f.TakesValue() {
			head += " " + f.Arg
		}
		heads[i] = head
		width = max(width, len(head))
	}
	for i, f := range flags {
		writeColumns(w, heads[i], f.Usage, width)
	}
}

func writeColumns(w io.Writer, left, right string, width int) {
	for i, line := range strings.Split(right, "\n") {
		if i > 0 {
			left = ""
		}
		fmt.Fprintf(w, "  %-*s  %s\n", width, left, line)
	}
}
