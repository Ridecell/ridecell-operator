{{ define "componentName" }}pulse{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/corporate{{ end }}
{{ template "ingress" . }}
