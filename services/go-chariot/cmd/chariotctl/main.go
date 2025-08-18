package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/bhouse1273/chariot-ecosystem/services/go-chariot/chariot"
)

func main() {
	script := flag.String("f", "", "path to .chariot script")
	flag.Parse()
	if *script == "" {
		fmt.Println("usage: chariotcli -f script.ch")
		os.Exit(1)
	}
	src, err := os.ReadFile(*script)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	rt := chariot.NewRuntime()
	chariot.RegisterAll(rt)
	out, err := rt.ExecProgram(string(src))
	if err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
	fmt.Println(out)
}
