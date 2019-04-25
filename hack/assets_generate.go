// +build ignore

package main

import (
	"log"
	"os"

	"github.com/Ridecell/ridecell-operator/pkg/controller/summon"
	"github.com/shurcooL/vfsgen"
)

func main() {
	err := vfsgen.Generate(summon.Templates, vfsgen.Options{
		PackageName:  os.Args[1],
		BuildTags:    "release",
		VariableName: "Templates",
		Filename:     "zz_generated.templates.go",
	})
	if err != nil {
		log.Fatalln(err)
	}
}
