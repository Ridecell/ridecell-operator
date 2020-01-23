{{ define "componentName" }}dispatch{{ end }}
{{ define "componentType" }}dispatch{{ end }}
{{ define "maxUnavailable" }}{{ if (gt (int .Instance.Spec.Replicas.Dispatch) 1) }}10%{{ else }}100%{{ end }}{{ end }}
{{ template "podDisruptionBudget" . }}
