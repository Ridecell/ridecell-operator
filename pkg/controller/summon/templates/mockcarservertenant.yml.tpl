apiVersion: summon.ridecell.io/v1beta1
kind: MockCarServerTenant
metadata:
  name: {{ .Instance.Name }}
  namespace: {{ .Instance.Namespace }}
spec:
  tenantHardwareType: {{ .Instance.Spec.MockTenantHardwareType }}
  callbackUrl: {{ .Extra.Vars.CallbackUrl }}
