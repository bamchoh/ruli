package main

import (
	"fmt"
	"os"

	"ruli"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: ruli <script>")
		os.Exit(1)
	}

	if err := ruli.New().RunFile(os.Args[1]); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
