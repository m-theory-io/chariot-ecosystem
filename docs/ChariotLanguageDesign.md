Chariot Language Design
=======================

Chariot is a descendant of early AI and functional languages such as
Lisp and Scheme. The original author spent many years developing
commercial applications in both Lisp and in various Microsoft languages,
such as Visual Basic .NET. C\# and Microsoft Visual C++. Chariot unifies
many of the virtues of those legacy languages, producing a tool that
makes creating secure, enterprise-worthy applications fun and relatively
friction-free.

Chariot Syntax
--------------

As a Lisp descendant, Chariot has a very simple syntax, inspired by the
idea of an s-expression, or symbolic expression. The Lisp syntax is so
simple, it has been called a syntax-free programming language, in that
you have a single syntactic construct, the s-expression:

```
(this (s expression) (could (be represented)) as (this tree))
```

In a Lisp function call, the first symbol of the s-expression is taken
to be the name of the function to call, while the remainder of the
symbols are considered arguments:

```
(setq x 5) // creates a variable x with the initial value of 5
```

Chariot simply puts the function name outside the parentheses:

```
setq(x, 5) // creates a local variable x with the initial value of 5
```

The next deviation from classic Lisp is in how Chariot handles branching
and iteration. In Lisp, a simple condition branching function is if:

```
(if (< x 5) (print true) (print false))
```

Chariot is very close, but introduces the notion of a code block,
enclosed by braces, to define the true and false expressions of a
branching or iterating function:

```
if(smaller(x, 5)) {
   print(true)
} else {
   print(false)
}
```

Is it more verbose than Lisp? Yes, but it is also arguably more readable
to programmers raised on imperative languages such as JavaScript or
C++. Also, note that else is treated as a keyword.\
\
For iteration, the same pattern applies. A Lisp while loop:

```
(while (< counter 5) ;; continue as long as 'counter' is less than 5
   (print counter) ;; Body: print the value of 'counter'
   (setq counter (1+ counter)) ;; Update: increment 'counter'
)
```

A Chariot while loop:

```
while(smaller(x, 5)) {
   print(x)
   setq(x add(x, 1))
}
```

The Chariot while loop also supports 2 additional keywords, break and
continue:

```
setq(x, 0)
while(smaller(x, 6)) {
   if(equal(x,3)) {
      continue // skips the increment
   }
   if(equal(x, 5)) {
      break // breaks out of the loop
   }
   setq(x, add(x, 1)) // increments the control variable
}
print(x) // prints 5
```

## Newlines and expression continuation

Statements are normally separated by newlines. Semicolons remain optional
explicit statement delimiters (including between statements on the same line),
not argument or array-element separators. Historical whitespace-separated
arguments remain supported; commas are recommended for clarity. Arrays require
commas between elements.

- A call's opening `(` must be on the same line as the end of its callee.
   Spaces and tabs are allowed: `f (42)` and `(f) (42)` are calls. After a
   completed expression, a newline before `(` ends that expression:
   `f` followed by newline then `(42)` means two expressions, **not** `f(42)`.
   The rule is identical for bare names, member reads, grouped expressions,
   literals, function literals, and results of earlier calls. Legacy split-name
   calls must move `(` onto the name's line; there is no runtime name lookup or
   builtin-name exception in the grammar.
- An explicit dot continues a member chain across newlines in either position:
   `object` followed by newline then `.method(42)`, or `object.` followed by
   newline then `method(42)`. Both are one call. The `(` must still share a line
   with `method`; a dot does not enable arbitrary newline-separated calls.
   Use `call(expression, args)` when a long callee needs a multiline call layout.
- Once `(` opens a call, arguments may span lines. Grouping and arrays may also
   span lines, and nested calls obey the same rules. Newlines do **not** close
   these delimiters: `array(1` followed by newline then `(2))` has two arguments
   under the legacy whitespace-argument syntax, not a call on `1`. Commas avoid
   this ambiguity. A newline *inside* a multiline string is not a boundary after
   that string.
- Required syntax is unaffected: `func` may be followed by its parameter list
   and body on later lines; control-flow headers/bodies, `else`, and declaration
   bodies such as `class(C)` followed by newline then `{ ... }` remain attached.
- `//` comments end at LF (also covering CRLF input); their following newline
   remains a boundary. Comment-only and blank lines do not hide a boundary or
   prevent explicit dot continuation. Block comments (`/* ... */`) are not
   supported and remain syntax errors. LF and CRLF have identical continuation
   behavior.

Go AST source renderers use newline-separated statements without inserting
mandatory semicolons, including before grouped calls and within control-flow
and function bodies. Explicit semicolons in input remain accepted but are not
preserved as AST nodes. This describes the Go parser/renderers, not a change to
the independent frontend/codegen implementations.

## Members, classes, and interfaces

Chariot also supports postfix member reads and calls such as `person.name`,
`person.hello()`, and `factory().hello()`. Writes remain explicit:
`setProp(person, "name", "Ada")`, not member assignment syntax.

The canonical references are:

- [Member notation and callable expressions](MemberNotation.md): precedence,
   receiver binding, null/missing members, host compatibility, and AST migration.
- [Script classes and structural interfaces](ClassesAndInterfaces.md): `class`,
   `field`, `method`, `new`, signatures, conformance, copying, and persistence.
- [Executable object examples](../examples/objects/README.md): self-contained
   interpreter inputs and expected final results (parent validation pending).
- [Implementation decisions and TODO](notes/Chariot%20Objects%20and%20Dot%20Notation%20TODO.md):
   approved implementation syntax, evidence boundaries, and deferred acceptance.

Classes use composition and explicit `self` receiver methods, not inheritance or
implicit `this`. Structural interfaces compare annotated methods without an
`implements` clause. Both `map(...)` and `mapValue(...)` currently provide named
construction attributes. These forms extend the function-oriented syntax rather
than introducing a general imperative assignment grammar.
