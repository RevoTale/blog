package main

import (
	"fmt"
	"os"

	"blog/framework/approutegen"
)

func main() {
	if err := approutegen.Run(); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "approutegen: %v\n", err)
		os.Exit(1)
	}
}
