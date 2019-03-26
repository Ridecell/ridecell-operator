kind: S3Bucket
apiVersion: aws.ridecell.io/v1beta1
metadata:
 name: {{ .Instance.Name }}-miv
 namespace: {{ .Instance.Namespace }}
spec:
 bucketName: ridecell-{{ .Instance.Name }}-miv
 region: {{ .Instance.Spec.AwsRegion }}
