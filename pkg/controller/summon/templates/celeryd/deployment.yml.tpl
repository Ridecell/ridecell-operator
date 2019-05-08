{{ define "componentName" }}celeryd{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "command" }}[python, "-m", celery, "-A", summon_platform, worker, "-l", info]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.WorkerReplicas }}{{ end }}
{{ define "memory_limit" }}2G{{ end }}
{{ template "deployment" . }}
