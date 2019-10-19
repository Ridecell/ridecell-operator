{{ define "componentName" }}daphne{{ end }}
{{ define "componentType" }}web{{ end }}
{{ template "podDisruptionBudget" . }}
