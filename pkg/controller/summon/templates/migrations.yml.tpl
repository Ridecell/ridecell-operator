kind: Migration
apiVersion: db.ridecell.io/v1beta1
metadata:
 name: {{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
spec:
 version: {{ .Instance.Spec.Version }}
 flavor: "{{ .Instance.Spec.Flavor }}"
 {{- if .Instance.Spec.EnableNewRelic -}}enableNewRelic: true{{- end -}}
