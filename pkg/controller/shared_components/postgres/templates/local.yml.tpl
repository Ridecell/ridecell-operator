apiVersion: acid.zalan.do/v1
kind: postgresql
metadata:
  name: {{ .Instance.Name }}-database
  namespace: {{ .Instance.Namespace }}
# This is filled in from the object.
spec: {}
