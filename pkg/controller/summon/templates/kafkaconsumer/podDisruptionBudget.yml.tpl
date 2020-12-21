{{ define "componentName" }}kafkaconsumer{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "maxUnavailable" }}{{ if (gt (int .Instance.Spec.Replicas.kafkaconsumer) 1) }}10%{{ else }}100%{{ end }}{{ end }}
{{ template "podDisruptionBudget" . }}