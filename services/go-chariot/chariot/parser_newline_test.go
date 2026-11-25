package chariot

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func TestNewlineExpressionBoundaries(t *testing.T) {
	for _, source := range []string{
		"42\n(func() { 7 })()",
		"name\n(7)", // Named calls follow the same rule as expression calls.
		"object.method\n(7)",
		"factory()\n(7)",
		"(factory)\n(7)",
		"[1, 2]\n(7)",
		"func() { 42 }\n(func() { 7 })()",
		"`first\nsecond`\n(7)",
		"42 // trailing comment\n// another comment\n(func() { 7 })()",
	} {
		for _, newline := range []string{"\n", "\r\n"} {
			source := strings.ReplaceAll(source, "\n", newline)
			t.Run(source, func(t *testing.T) {
				node := dotParse(t, source)
				if len(node.Stmts) != 2 {
					t.Fatalf("got %d statements, want 2: %s", len(node.Stmts), node.ToString())
				}
			})
		}
	}
}

func TestNewlineContinuationAndArguments(t *testing.T) {
	for _, source := range []string{
		"(func(x) { x }) (42)",
		"(func() { func(x) { x } })()(42)",
		"(func() { 42\n(func() { 42 })() })()",
		"mapValue(\n\"f\", func(x) { x }\n).f(\n42\n)",
		"mapValue(\"f\", func(x) { x })\n.f(42)",
		"mapValue(\"f\", func(x) { x }).\nf(42)",
		"mapValue(\"f\", func(x) { x }) // continue explicitly\n. // member follows\nf(42)",
		"mapValue(\"f\", func(x) { mapValue(\"n\", x) })\n.f(42)\n.n",
		"(\nfunc\n(x)\n{\nx\n}\n)(\n42\n)",
		"setq(f, func(x) { x })\nf (42)",
		"setq(f, func(x) { x })\nf(\n(42)\n)",
		"setq(f, func(x) { x })\nf // arguments do not attach\n(42)",
		"1; (func() { 42 })()", // Explicit same-line statement delimiter.
		"(func(x) { x })(42\n)",
		"setq(f, func(x, y) { y })\nf(1 42)", // Legacy whitespace-separated args.
	} {
		for _, newline := range []string{"\n", "\r\n"} {
			source := strings.ReplaceAll(source, "\n", newline)
			t.Run(source, func(t *testing.T) {
				if got := dotExec(t, classRuntime(), source); got != Number(42) {
					t.Fatalf("got %v, want 42", got)
				}
			})
		}
	}
	node := dotParse(t, "array(1\n(2)\n[3,\n4],\narray(5\n6))")
	if got := len(node.Stmts[0].(*FuncCall).Args); got != 4 {
		t.Fatalf("got %d arguments, want 4", got)
	}
	got := dotExec(t, classRuntime(), "array(1\n(2)\n[3,\n4],\narray(5\n6))").(*ArrayValue)
	if got.Length() != 4 || got.Get(0) != Number(1) || got.Get(1) != Number(2) ||
		!reflect.DeepEqual(got.Get(2).(*ArrayValue).Elements, []Value{Number(3), Number(4)}) ||
		!reflect.DeepEqual(got.Get(3).(*ArrayValue).Elements, []Value{Number(5), Number(6)}) {
		t.Fatalf("nested multiline arguments = %#v", got)
	}
	for _, source := range []string{"array(1; 2)", "[1; 2]", "object.;method()", "1 /* block\ncomment */\n(2)"} {
		if _, err := NewParser(source).parseProgram(); err == nil {
			t.Fatalf("accepted unsupported syntax %q", source)
		}
	}
}

func TestNewlineLexerTrivia(t *testing.T) {
	for _, test := range []struct {
		source string
		want   bool
	}{
		{"`first\nsecond`(42)", false},
		{"\"first\nsecond\"(42)", false},
		{"name (42)", false},
		{"name\t(42)", false},
		{"name\n(42)", true},
		{"name\r\n(42)", true},
		{"name // comment\r\n// another\r\n(42)", true},
	} {
		lexer := NewLexer(test.source)
		lexer.Next()
		if tok := lexer.Next(); tok.Type != TOK_LPAREN || tok.NewlineBefore != test.want {
			t.Errorf("%q: got %+v, want '(' with NewlineBefore=%v", test.source, tok, test.want)
		}
	}
}

func TestNewlineSourceRoundTrips(t *testing.T) {
	for _, source := range []string{
		"1\n(func() { 42 })()",
		"if(true) { 1\n(func() { 42 })() }",
		"if(false) { 0 }\nelse\n{ 1\n(func() { 42 })() }",
		"setq(n, 0)\nwhile(equal(n, 0)) { setq(n, 1)\n(func() { 42 })() }",
		"switch(1) { case(1) { 1\n(func() { 42 })() } default() { 0 } }",
		"switch(2) { case(1) { 0 } default() { 1\n(func() { 42 })() } }",
		"(func() { 1\n(func() { 42 })() })()",
		"class(C)\n{ field(n, \"N\", 42) }\nsetq(c, new(C))\n1\n(func() { c.n })()",
	} {
		t.Run(source, func(t *testing.T) {
			node := dotParse(t, source)
			for _, rendered := range []string{source, node.ToString(), PrettyPrintNode(node, "")} {
				if strings.Contains(rendered, ";") {
					t.Fatalf("renderer inserted semicolons: %s", rendered)
				}
				if got := dotExec(t, classRuntime(), rendered); got != Number(42) {
					t.Fatalf("rendered %s: got %v, want 42", rendered, got)
				}
			}
		})
	}
}

func TestNewlineObjectExamplesWithoutSemicolons(t *testing.T) {
	wants := map[string][]Value{
		"member-calls.ch":  {Number(42), Number(42), Number(42), Bool(true)},
		"classes.ch":       {Str("Grace"), Str("Grace"), Str("Ada"), Number(1), Number(0), Number(42), Str("Grace")},
		"interfaces.ch":    {Bool(true), Number(42), Number(42), Str("F"), Str("O")},
		"serialization.ch": {Str("Ada"), Str("Ada"), Str("Grace"), Str("O")},
	}
	for file, want := range wants {
		t.Run(file, func(t *testing.T) {
			data, err := os.ReadFile(filepath.Join("..", "..", "..", "examples", "objects", file))
			if err != nil {
				t.Fatal(err)
			}
			source := strings.ReplaceAll(string(data), ";", "")
			node := dotParse(t, source)
			for _, rendered := range []string{source, node.ToString(), PrettyPrintNode(node, "")} {
				if strings.Contains(rendered, ";") {
					t.Fatalf("renderer inserted semicolons: %s", rendered)
				}
				rt := classRuntime()
				RegisterETLFunctions(rt)
				RegisterJSON(rt)
				got := dotExec(t, rt, rendered)
				array, ok := got.(*ArrayValue)
				if !ok || !reflect.DeepEqual(array.Elements, want) {
					t.Fatalf("got %#v, want %#v", got, want)
				}
			}
		})
	}
}
