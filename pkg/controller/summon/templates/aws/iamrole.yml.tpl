kind: IAMRole
apiVersion: aws.ridecell.io/v1beta1
metadata:
 name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
 namespace: {{ .Instance.Namespace }}
spec:
 roleName: summon-platform-{{ .Instance.Name }}
 inlinePolicies:
   allow_s3: |
            {
               "Version": "2012-10-17",
               "Statement": [
                {
                   "Effect": "Allow",
                   "Action": [
                      "s3:ListBucket"
                    ],
                   "Resource": "arn:aws:s3:::ridecell-{{ .Instance.Name }}-static"
                },
                {
                   "Effect": "Allow",
                   "Action": [
                      "s3:GetObject",
                      "s3:DeleteObject",
                      "s3:PutObject",
                      "s3:PutObjectAcl"
                    ],
                   "Resource": "arn:aws:s3:::ridecell-{{ .Instance.Name }}-static/*"
                }
              ]
            }
{{if .Extra.optimusBucketName}}
   allow_s3_optimus: |
            {
               "Version": "2012-10-17",
               "Statement": [
                {
                   "Effect": "Allow",
                   "Action": [
                      "s3:GetObject"
                    ],
                   "Resource": "arn:aws:s3:::{{ .Extra.optimusBucketName }}/*"
                }
              ]
            }
{{end}}
   allow_s3_miv: |
            {
               "Version": "2012-10-17",
               "Statement": [
                 {
                    "Effect": "Allow",
                    "Action": [
                       "s3:ListBucket"
                     ],
                    "Resource": "arn:aws:s3:::{{ .Extra.mivBucket }}"
                 },
                 {
                    "Effect": "Allow",
                    "Action": [
                       "s3:GetObject",
                       "s3:DeleteObject",
                       "s3:PutObject",
                       "s3:PutObjectAcl"
                     ],
                    "Resource": "arn:aws:s3:::{{ .Extra.mivBucket }}/*"
                 }
               ]
            }
   allow_sqs: |
            {
              "Version": "2012-10-17",
              "Statement": {
                "Sid": "",
                "Effect": "Allow",
                "Action": [
                  "sqs:SendMessageBatch",
                  "sqs:SendMessage",
                  "sqs:CreateQueue"
                ],
                "Resource": [
                  "arn:aws:sqs:us-west-2:{{ .Extra.accountId }}:{{ .Instance.Spec.SQSQueue }}",
                  "arn:aws:sqs:eu-central-1:{{ .Extra.accountId }}:{{ .Instance.Spec.SQSQueue }}",
                  "arn:aws:sqs:ap-south-1:{{ .Extra.accountId }}:{{ .Instance.Spec.SQSQueue }}"
                ]
              }
            }
 permissionsBoundaryArn: {{ .Extra.permissionsBoundaryArn }}
