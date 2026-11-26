package main

import (
	"fmt"
	"log"

	"github.com/m-theory-io/chariot-ecosystem/services/go-chariot/chariot"
)

func main() {
	rt := chariot.NewRuntime()
	// EmbeddedProfile omits app-server categories (SQL, Couchbase, listeners,
	// auth, RBAC, signals). Use chariot.RegisterAll(rt) for the full set, or
	// chariot.Without(...) to exclude specific categories.
	chariot.RegisterAll(rt, chariot.EmbeddedProfile())

	// Optionally bind host objects
	// rt.BindObject("db", dbClient)

	script := `
        declare(n, 'N', 0)
        while(smallerEq(n, 5)) {
            setq(n, add(n, 1))
        }
        append('Result: n=', n)
    `

	result, err := rt.ExecProgram(script)
	if err != nil {
		log.Fatal(err)
	}
	fmt.Println(result)
}
