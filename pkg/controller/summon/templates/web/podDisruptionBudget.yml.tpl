{{ define "componentName" }}web{{ end }}
{{ define "componentType" }}web{{ end }}
{{ define "maxUnavailable" }}{{ if (gt (int .Instance.Spec.Replicas.Web) 1) }}10%{{ else }}0{{ end }}{{ end }}
{{ template "podDisruptionBudget" . }}