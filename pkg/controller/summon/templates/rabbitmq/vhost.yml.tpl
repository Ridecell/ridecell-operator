apiVersion: db.ridecell.io/v1beta1
kind: RabbitmqVhost
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
spec:
  vhostName: {{ .Instance.Name }}
  policies:
    HA:
      pattern: ^(?!amq\.).*
      apply-to: queues
      priority: -10
      definition: |
        ha-mode: exactly
        ha-params: 2
        ha-sync-mode:	automatic
    Federate:
      pattern: ^celery$
      apply-to: queues
      priority: 100
      definition: |
        ha-mode: exactly
        ha-params: 2
        ha-sync-mode: automatic
