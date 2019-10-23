{{ define "service" }}
kind: Service
apiVersion: v1
metadata:
  name: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
  namespace: {{ .Instance.Namespace }}
  labels:
    app.kubernetes.io/name: {{ block "componentName" . }}{{ end }}
    app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}
    app.kubernetes.io/version: {{ .Instance.Spec.Version }}
    app.kubernetes.io/component: {{ block "componentType" . }}{{ end }}
    app.kubernetes.io/part-of: {{ .Instance.Name }}
    app.kubernetes.io/managed-by: summon-operator
    {{ block "extraLabels" . }}{{ end -}}
  annotations:
{{ block "extraAnnotations" . }}{{ end -}}
spec:
  selector:
    {{ block "selectors" . }}{app.kubernetes.io/instance: {{ .Instance.Name }}-{{ block "componentName" . }}{{ end }}}{{ end }}
  ports: {{ block "servicePorts" . }}[{protocol: TCP, port: 8000}]{{ end }}
{{ end }}
