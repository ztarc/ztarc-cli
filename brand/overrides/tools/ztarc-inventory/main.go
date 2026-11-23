package main

import (
	"encoding/json"
	"github.com/fosrl/cli/cmd"
	"github.com/fosrl/cli/internal/policy"
	"github.com/spf13/cobra"
	"os"
)

type entry struct {
	Command string `json:"command"`
	Stage   string `json:"stage"`
	Status  string `json:"status"`
}

func main() {
	root, err := cmd.RootCommand(false)
	if err != nil {
		panic(err)
	}
	all := []entry{}
	var walk func(*cobra.Command)
	walk = func(c *cobra.Command) {
		stage, status := "A", "not_tested"
		if policy.Guard(c) != nil {
			stage, status = "B", "deferred"
			for ancestor := c; ancestor != nil; ancestor = ancestor.Parent() {
				if ancestor.Name() == "apply" || ancestor.Name() == "configure" {
					stage = "C"
				}
				if ancestor.Name() == "companion" {
					stage = "D"
				}
			}
		}
		all = append(all, entry{c.CommandPath(), stage, status})
		for _, child := range c.Commands() {
			walk(child)
		}
	}
	walk(root)
	e := json.NewEncoder(os.Stdout)
	e.SetIndent("", "  ")
	e.Encode(all)
}
