{{ define "verticalPodAutoscaler" }}
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" .}}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    {{/* creator and source needed for goldilocks dashboard to pick vpa up */ -}}
    creator: "Fairwinds"
    source: "goldilocks"
spec:
  targetRef: {{ block "controller" . }}{{ end }}
  updatePolicy:
    updateMode: {{ block "updateMode" . }}"Off"{{ end }}
  {{/* Leaving ResourcePolicy blank so autoscaler can compute recommended */ -}}
{{ end }}
