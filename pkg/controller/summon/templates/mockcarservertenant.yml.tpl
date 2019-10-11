apiVersion: summon.ridecell.io/v1beta1
kind: MockCarServerTenant
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
spec:
  TenantHardwareType: {{ .Instance.Spec.MockTenantHardwareType }}
