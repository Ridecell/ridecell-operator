#!/bin/bash
set -euo pipefail
IFS=$'\n\t'

prefix="$(mktemp "${TMPDIR:-/tmp}/assets_generate_XXXXXX")"
tmp="$(mktemp "$prefix.go")"
trap "rm $prefix $tmp" EXIT
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
