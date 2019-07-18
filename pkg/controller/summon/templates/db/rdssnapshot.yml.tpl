kind: RDSSnapshot
apiVersion: db.ridecell.io/v1beta1
metadata:
 name: {{ .Instance.Name }}-{{ .Instance.Spec.Version }}
 namespace: {{ .Instance.Namespace }}
spec:
 RDSInstanceID: {{ .Extra.rdsInstanceName }}
 ttl: {{ .Instance.Spec.Backup.TTL.Duration }}
