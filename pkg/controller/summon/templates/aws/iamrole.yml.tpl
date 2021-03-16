apiVersion: identity.aws.crossplane.io/v1beta1
kind: IAMRole
metadata:
  name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
  annotations:
    crossplane.io/external-name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
spec:
  roleName: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
  assumeRolePolicyDocument: {{ .Extra.assumeRolePolicyDocument | toJson }}
  permissionsBoundaryArn: {{ .Extra.permissionsBoundaryArn }}
