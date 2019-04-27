apiVersion: db.ridecell.io/v1beta1
kind: RDSInstance
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
# This is filled in from the object.
spec: {}
