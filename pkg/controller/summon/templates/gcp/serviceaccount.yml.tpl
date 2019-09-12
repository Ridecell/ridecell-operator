apiVersion: gcp.ridecell.io/v1beta1
kind: ServiceAccount
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
spec:
  project: {{ .Instance.Spec.GCPProject }}
