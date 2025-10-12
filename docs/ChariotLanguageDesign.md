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

That's it -- everything else in Chariot is either a function, a variable
reference (i. e., unquoted string), or a literal value (naked numbers
and quoted strings). Chariot is almost as simple as Lisp.
