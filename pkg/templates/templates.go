/*
Copyright 2018-2019 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package templates

import (
	"bytes"
	"net/http"
	"path"
	"text/template"

	// "github.com/golang/glog"
	"github.com/Masterminds/sprig"
	"github.com/shurcooL/httpfs/path/vfspath"
	"github.com/shurcooL/httpfs/vfsutil"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/scheme"
)

func parseTemplate(fs http.FileSystem, filename string) (*template.Template, error) {
	// Create a template object.
	tmpl := template.New(path.Base(filename)).Funcs(sprig.TxtFuncMap())

	// Parse any helpers if present.
	helpers, err := vfspath.Glob(fs, "helpers/*.tpl")
	if err != nil {
		return nil, err
	}
	for _, helperFilename := range helpers {
		fileBytes, err := vfsutil.ReadFile(fs, helperFilename)
		if err != nil {
			return nil, err
		}

		_, err = tmpl.Parse(string(fileBytes))
		if err != nil {
			return nil, err
		}
	}

	// Parse the main template.
	fileBytes, err := vfsutil.ReadFile(fs, filename)
	if err != nil {
		return nil, err
	}

	_, err = tmpl.Parse(string(fileBytes))
	if err != nil {
		return nil, err
	}

	return tmpl, nil
}

func renderTemplate(tmpl *template.Template, data interface{}) (string, error) {
	var buffer bytes.Buffer
	err := tmpl.Execute(&buffer, data)
	if err != nil {
		return "", err
	}
	return buffer.String(), nil
}

func parseObject(rawObject string) (runtime.Object, error) {
	// Parse the rendered data into an object. The caller has to cast it from a
	// runtime.Object into the correct type.
	obj, _, err := scheme.Codecs.UniversalDeserializer().Decode([]byte(rawObject), nil, nil)
	if err != nil {
		return nil, err
	}
	return obj, nil
}

func Get(fs http.FileSystem, filename string, data interface{}) (runtime.Object, error) {
	tmpl, err := parseTemplate(fs, filename)
	if err != nil {
		return nil, err
	}
	out, err := renderTemplate(tmpl, data)
	if err != nil {
		return nil, err
	}
	obj, err := parseObject(out)
	if err != nil {
		return nil, err
	}
	return obj, nil
}
