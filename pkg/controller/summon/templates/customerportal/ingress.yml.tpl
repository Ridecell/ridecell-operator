{{ define "componentName" }}customerportal{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/reserve{{ end }}
{{ template "ingress" . }}
