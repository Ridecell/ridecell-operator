{{ define "componentName" }}tripshare{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/tripshare{{ end }}
{{ template "ingress" . }}
