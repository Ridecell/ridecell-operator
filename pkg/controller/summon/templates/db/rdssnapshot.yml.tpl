kind: RDSSnapshot
apiVersion: db.ridecell.io/v1beta1
metadata:
 name: {{ .Instance.Name }}-{{ .Instance.Spec.Version | replace "_" "-" }}
 namespace: {{ .Instance.Namespace }}
spec:
 rdsInstanceID: {{ .Extra.rdsInstanceName }}
 ttl: {{ .Instance.Spec.Backup.TTL.Duration }}
