{{ define "hpa"}}
apiVersion: autoscaling/v2beta2
kind: HorizontalPodAutoscaler
metadata:
  name: {{ .Instance.Name }}-{{block "componentName" . }}{{ end }}-hpa
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
spec:
  scaleTargetRef: {{ block "target" . }}{{ end }}
  minReplicas: {{ block "minReplicas" . }}1{{ end }}
  maxReplicas: {{ block "maxReplicas" . }}10{{ end }}
  metrics:
  - type: External
    external:
      metric: {{ block "metric" . }}{{ end }}
      target: {{ block "mTarget" . }}{{ end }}
{{ end }}