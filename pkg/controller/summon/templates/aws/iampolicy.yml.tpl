apiVersion: identity.aws.crossplane.io/v1alpha1
kind: IAMPolicy
metadata:
  name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
  annotations:
    crossplane.io/external-name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
spec:
  forProvider:
  name: summon-platform-{{ .Instance.Spec.Environment }}-{{ .Instance.Name }}
  document: |
            {
               "Version": "2012-10-17",
               "Statement": [
                {
                   "Effect": "Allow",
                   "Action": [
                      "s3:ListBucket"
                    ],
                    "Resource": [
                      "arn:aws:s3:::{{ .Extra.mivBucket }}/*",
                      "arn:aws:s3:::ridecell-{{ .Instance.Name }}-static/*"
                    ]
                },
                {
                   "Effect": "Allow",
                   "Action": [
                      "s3:GetObject",
                      "s3:DeleteObject",
                      "s3:PutObject",
                      "s3:PutObjectAcl"
                    ],
                   "Resource": [
                     "arn:aws:s3:::{{ .Extra.mivBucket }}/*",
                     "arn:aws:s3:::ridecell-{{ .Instance.Name }}-static/*"
                   ]
                },
                {
                   "Effect": "Allow",
                   "Action": [
                      "s3:GetObject"
                    ],
                   "Resource": "arn:aws:s3:::{{ .Extra.optimusBucketName }}/*"
                },
                {
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
                },
                {
                      "Sid": "",
                      "Effect": "Allow",
                      "Action": [
                         "ses:SendTemplatedEmail"
                      ],
                      "Resource": "*"
                }
              ]
            }
