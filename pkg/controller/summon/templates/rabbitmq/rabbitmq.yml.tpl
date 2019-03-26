apiVersion: db.ridecell.io/v1beta1
kind: RabbitmqUser
metadata:
 name: {{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
spec:
 username: {{ .Instance.Name }}-user
 tags: policymaker
 connection:
   username: ridecell-operator
   passwordSecretRef: ridecell-operator-rabbitmq.credentials
   host: {{ .Instance.Spec.Database.Connection.Host }}
