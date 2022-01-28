package main

import (
	"fmt"
	"github.com/hobbyfarm/gargantua/pkg/crd"
	"os"
)

func main() {
	if len(os.Args) > 2 && os.Args[1] == "crds" {
		fmt.Println("Writing CRDs to", os.Args[2])
		if err := crd.WriteFile(os.Args[2]); err != nil {
			panic(err)
		}
		return
	}

}
