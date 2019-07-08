kind: RDSSnapshot
apiVersion: db.ridecell.io/v1beta1
metadata:
 name: {{ .Extra.backupName }}
 namespace: {{ .Instance.Namespace }}
spec:
 RDSInstanceID: {{ .Extra.rdsInstanceName }}
 ttl: {{ .Instance.Spec.Backup.TTL }}
