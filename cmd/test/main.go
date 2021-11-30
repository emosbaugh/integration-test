package main

import (
	"log"

	"github.com/spf13/cobra"
)

func main() {
	cmd := &cobra.Command{
		Use:   "test",
		Short: "my test program",
	}
	err := cmd.Execute()
	if err != nil {
		log.Fatal(err)
	}
}
