apiVersion: monitoring.ridecell.io/v1beta1
kind: Monitor
metadata:
  name: {{ .Instance.Name }}-monitoring
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: monitoring
    app.kubernetes.io/instance: {{ .Instance.Name }}-monitoring
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: monitoring
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  servicename: {{ .Instance.Name }}
  notify:
    slack: [{{ .Instance.Spec.Notifications.SlackChannel | quote }}]
    {{ if .Instance.Spec.Notifications.Pagerdutyteam -}}   
    pagerdutyteam: {{ .Instance.Spec.Notifications.Pagerdutyteam | quote }}
    {{ end -}}
  metricAlertRules:
    - alert: Newrelic error {{ .Instance.Name }}
      expr: newrelic_application_error_rate{appname="{{ .Instance.Name }}-summon-platform"} >= 1
      for: 5m
      labels:
        severity: critical
        servicename: {{ .Instance.Name }}
      annotations:
        summary: Newrelic error % greater than 1 for {{ .Instance.Name }}
    - alert: uptime check failed for {{ .Instance.Name }}-web
      expr: probe_success{name="{{ .Instance.Name }}-web"} == 0
      for: 5m
      labels:
        severity: critical
        servicename: {{ .Instance.Name }}
      annotations:
        summary: prober not able to reach {{ .Instance.Name }}-web
    - alert: Pods are not running
      expr: kube_pod_container_status_running{namespace={{ .Instance.Namespace | quote }}, pod=~"{{ .Instance.Name }}.*" ,pod!~ "{{ .Instance.Name }}-migrations-.*"} 
      for: 3m
      labels:
        severity: critical
        servicename: {{ .Instance.Name }}
      annotations:
        summary: "{{ $labels.pod }} pod is not running."
    - alert: Memory Critical
      expr: container_memory_usage_bytes{namespace={{ .Instance.Namespace | quote }}, pod=~"{{ .Instance.Name }}-.*" }  / on(pod, container) kube_pod_container_resource_limits_memory_bytes{namespace={{ .Instance.Namespace | quote }}, pod=~"{{ .Instance.Name }}-.*"}  * 100 > 80
      for: 10m
      labels:
        severity: info
        servicename: {{ .Instance.Name }}
      annotations:
        summary: "{{ $labels.pod }}/{{ $labels.container }} pod utilized {{ $value }}%  memory"
    - alert: Too Many Messages In Queue
      expr: rabbitmq_queue_messages_ready{queue="celery", vhost="{{ .Instance.Name }}"} > 100
      for: 5m
      labels:
        severity: info
        servicename: {{ .Instance.Name }}
      annotations:
        summary: "Too many messages in {{ .Instance.Name }}/celery queue"
    - alert: No Consumers
      expr: rabbitmq_queue_consumers{vhost="{{ .Instance.Name }}"} == 0
      for: 5m
      labels:
        severity: info
        servicename: {{ .Instance.Name }}
      annotations:
        summary: "No consumers for {{ .Instance.Name }}/celery queue. Check celery pods"