apiVersion: db.ridecell.io/v1beta1
kind: RabbitmqUser
metadata:
 name: {{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
spec:
 username: {{ .Instance.Name }}-user
 tags: policymaker
 insecureSkip: false
