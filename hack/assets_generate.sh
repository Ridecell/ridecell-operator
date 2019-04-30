#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

tmp="assets_generate.$$.go"
trap "rm $tmp" EXIT
cat <<EOH > "$tmp"
// +build ignore

package main

import (
  "log"

  "github.com/Ridecell/ridecell-operator/pkg/$1"
  "github.com/shurcooL/vfsgen"
)

func main() {
  err := vfsgen.Generate($2.Templates, vfsgen.Options{
    PackageName:  "$2",
    BuildTags:    "release",
    VariableName: "Templates",
    Filename:     "zz_generated.templates.go",
  })
  if err != nil {
    log.Fatalln(err)
  }
}
EOH

go run "$tmp"
