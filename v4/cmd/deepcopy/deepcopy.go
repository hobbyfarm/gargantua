package main

import (
	"github.com/hobbyfarm/gargantua/v4/pkg/deepcopy"
	"os"
)

// Shamelessly stolen from github.com/acorn-io/baaah
func main() {
	deepcopy.Deepcopy(os.Args[1:]...)
}
