{{ define "componentName" }}channelworker{{ end }}
{{ define "componentType" }}worker{{ end }}
{{ define "command" }}[python, manage.py, runworker, "-v2", "--threads", "2"]{{ end }}
{{ define "replicas" }}{{ .Instance.Spec.Replicas.ChannelWorker | default 0 }}{{ end }}
{{ define "resources" }}{requests: {memory: "250M", cpu: "5m"}, limits: {memory: "300M"}}{{ end }}
{{ template "deployment" . }}
