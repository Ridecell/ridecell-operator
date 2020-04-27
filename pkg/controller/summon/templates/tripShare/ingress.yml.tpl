{{ define "componentName" }}tripshare{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "ingressPath" }}/trip_share{{ end }}
{{ template "ingress" . }}
