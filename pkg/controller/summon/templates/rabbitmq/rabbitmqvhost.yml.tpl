apiVersion: db.ridecell.io/v1beta1
kind: RabbitmqVhost
metadata:
 name: {{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
spec:
 vhostName: {{ .Instance.Name }}
