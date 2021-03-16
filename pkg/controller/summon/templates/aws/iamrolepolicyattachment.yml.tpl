apiVersion: identity.aws.crossplane.io/v1beta1
kind: IAMRolePolicyAttachment
metadata:
  name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
spec:
  forProvider:
    policyArnRef:
      name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
    roleNameRef:
      name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
