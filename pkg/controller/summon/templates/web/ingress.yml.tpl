{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/{{ end }}
{{ template "ingress" . }}
