{{ define "componentName" }}pulse{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/operations{{ end }}
{{ template "ingress" . }}
