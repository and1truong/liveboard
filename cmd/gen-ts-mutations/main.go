// Command gen-ts-mutations generates the TypeScript MutationOp tagged
// union from the Go-side mutation registry in internal/board.
//
// The Go file internal/board/mutation.go is the source of truth for every
// mutation variant — its struct fields, json tags, and types. This program
// reflects on a zero-value of each registered op and emits an equivalent
// TypeScript file. Running `make codegen` keeps the two in sync; `make
// verify-codegen` fails CI if the generator output drifts from disk.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"reflect"
	"slices"
	"sort"
	"strings"

	"github.com/and1truong/liveboard/internal/board"
)

func main() {
	out := flag.String("out", "", "output TypeScript file path")
	flag.Parse()
	if *out == "" {
		fmt.Fprintln(os.Stderr, "usage: gen-ts-mutations -out <path>")
		os.Exit(2)
	}

	rendered, err := render(board.RegisteredOpZeroValues())
	if err != nil {
		fmt.Fprintf(os.Stderr, "render: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(*out, rendered, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "write %s: %v\n", *out, err)
		os.Exit(1)
	}
}

// render produces the contents of mutations.gen.ts for a given registry view.
func render(variants map[string]any) ([]byte, error) {
	names := make([]string, 0, len(variants))
	for k := range variants {
		names = append(names, k)
	}
	sort.Strings(names)

	var b bytes.Buffer
	b.WriteString("// AUTO-GENERATED FROM internal/board/mutation.go.\n")
	b.WriteString("// Run `make codegen` to regenerate. Do not edit by hand.\n")
	b.WriteString("\n")
	b.WriteString("import type { BoardSettings } from './types.js'\n")
	b.WriteString("\n")

	for _, name := range names {
		zero := variants[name]
		t := reflect.TypeOf(zero)
		if t.Kind() == reflect.Ptr {
			t = t.Elem()
		}
		if t.Kind() != reflect.Struct {
			return nil, fmt.Errorf("variant %q: expected pointer-to-struct, got %v", name, reflect.TypeOf(zero))
		}
		if err := emitInterface(&b, name, t); err != nil {
			return nil, err
		}
		b.WriteString("\n")
	}

	b.WriteString("export type MutationOp =\n")
	for _, name := range names {
		fmt.Fprintf(&b, "  | %s\n", interfaceName(name))
	}

	return b.Bytes(), nil
}

func emitInterface(b *bytes.Buffer, opType string, t reflect.Type) error {
	fmt.Fprintf(b, "export interface %s {\n", interfaceName(opType))
	fmt.Fprintf(b, "  type: '%s'\n", opType)

	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		jsonTag := f.Tag.Get("json")
		if jsonTag == "" || jsonTag == "-" {
			continue
		}
		parts := strings.Split(jsonTag, ",")
		fieldName := parts[0]
		if fieldName == "" {
			continue
		}
		omit := slices.Contains(parts[1:], "omitempty")

		tsType, err := goTypeToTS(f.Type)
		if err != nil {
			return fmt.Errorf("%s.%s: %w", t.Name(), f.Name, err)
		}

		opt := ""
		if omit {
			opt = "?"
			// Pointer fields with omitempty model "absent | null | value"
			// in JSON: surface that as `T | null` on the TS side, matching
			// the prior hand-written shape.
			if f.Type.Kind() == reflect.Ptr {
				tsType += " | null"
			}
		}
		fmt.Fprintf(b, "  %s%s: %s\n", fieldName, opt, tsType)
	}
	b.WriteString("}\n")
	return nil
}

// goTypeToTS maps a Go reflect.Type to its TypeScript equivalent for the
// subset of shapes the mutation registry uses. Adding a new shape here is
// safer than letting the generator fall through to an unsupported case.
func goTypeToTS(t reflect.Type) (string, error) {
	switch t.Kind() {
	case reflect.String:
		return "string", nil
	case reflect.Bool:
		return "boolean", nil
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
		reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
		reflect.Float32, reflect.Float64:
		return "number", nil
	case reflect.Slice:
		elem, err := goTypeToTS(t.Elem())
		if err != nil {
			return "", err
		}
		return elem + "[]", nil
	case reflect.Ptr:
		return goTypeToTS(t.Elem())
	case reflect.Struct:
		// One concrete cross-package reference today: models.BoardSettings.
		// The generated file imports it from types.ts at the top.
		switch t.Name() {
		case "BoardSettings":
			return "BoardSettings", nil
		}
	}
	return "", fmt.Errorf("unsupported Go type: %s (kind=%s)", t.String(), t.Kind())
}

// interfaceName converts a snake_case mutation type ("add_card") to a
// PascalCase TS interface name ("AddCardOp").
func interfaceName(opType string) string {
	parts := strings.Split(opType, "_")
	for i, p := range parts {
		if p == "" {
			continue
		}
		parts[i] = strings.ToUpper(p[:1]) + p[1:]
	}
	return strings.Join(parts, "") + "Op"
}
