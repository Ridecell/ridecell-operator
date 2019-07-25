{
  "apiVersion": "extensions/v1beta1",
  "kind": "Ingress",
  "metadata": {
    "name": {{ .Instance.Name | toJson }},
    "namespace": {{ .Instance.Namespace | toJson }},
    "labels": {{ .Instance.Labels | toJson }},
    "annotations": {{ .Instance.Annotations | toJson }}
  },
  "spec": {{ .Instance.Spec | toJson }}
}
