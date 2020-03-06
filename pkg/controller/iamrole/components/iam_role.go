/*
Copyright 2020 Ridecell, Inc.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package components

import (
	"bytes"
	"encoding/json"
	"net/url"
	"os"
	"reflect"
	"text/template"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	corev1 "k8s.io/api/core/v1"
)

const iamRoleFinalizer = "iamrole.finalizer"

type iamRoleComponent struct {
	iamAPI iamiface.IAMAPI
}

type templatingData struct {
	Region string
}

func NewIAMRole() *iamRoleComponent {
	sess := session.Must(session.NewSession())
	iamService := iam.New(sess)
	return &iamRoleComponent{iamAPI: iamService}
}

func (comp *iamRoleComponent) InjectIAMAPI(iamapi iamiface.IAMAPI) {
	comp.iamAPI = iamapi
}

func (_ *iamRoleComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *iamRoleComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *iamRoleComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.IAMRole)

	// do the template thing on all the stuff
	templateData := templatingData{
		Region: os.Getenv("AWS_REGION"),
	}

	roleName, err := templateData.parseField(instance.Spec.RoleName)
	if err != nil {
		return components.Result{}, err
	}

	assumePolicyDocument, err := templateData.parseField(instance.Spec.AssumeRolePolicyDocument)
	if err != nil {
		return components.Result{}, err
	}

	inlinePolicies := map[string]string{}
	for policyName, policyValue := range instance.Spec.InlinePolicies {
		parsedPolicy, err := templateData.parseField(policyValue)
		if err != nil {
			return components.Result{}, err
		}
		inlinePolicies[policyName] = parsedPolicy
	}

	// if object is not being deleted
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Is our finalizer attached to the object?
		if !helpers.ContainsFinalizer(iamRoleFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(iamRoleFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iamrole: failed to update instance while adding finalizer")
			}
			return components.Result{Requeue: true}, nil
		}
	} else {
		if helpers.ContainsFinalizer(iamRoleFinalizer, instance) {
			if flag := instance.Annotations["ridecell.io/skip-finalizer"]; flag != "true" && os.Getenv("ENABLE_FINALIZERS") == "true" {
				result, err := comp.deleteDependencies(roleName)
				if err != nil {
					return result, err
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(iamRoleFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iamrole: failed to update instance while removing finalizer")
			}
			return components.Result{}, nil
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// check assumeRolePolicyDocument for valid JSON
	// inlinepolicies is checked later in UnMarshal
	if !json.Valid([]byte(assumePolicyDocument)) {
		return components.Result{}, errors.New("iam_role: assume role trust policy contains invalid json")
	}

	// Try to get our role, if it can't be found create it
	var role *iam.Role
	getRoleOutput, err := comp.iamAPI.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(aerr, "iam_role: failed to get role")
		}
		// If role does not exist create it
		createRoleOutput, err := comp.iamAPI.CreateRole(&iam.CreateRoleInput{
			RoleName:                 aws.String(roleName),
			PermissionsBoundary:      aws.String(instance.Spec.PermissionsBoundaryArn),
			AssumeRolePolicyDocument: aws.String(assumePolicyDocument),
			Tags: []*iam.Tag{
				&iam.Tag{
					Key:   aws.String("ridecell-operator"),
					Value: aws.String("True"),
				},
				&iam.Tag{
					Key:   aws.String("Kiam"),
					Value: aws.String("true"),
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: failed to create role")
		}
		role = createRoleOutput.Role
	} else {
		// If getRole did not return an error
		role = getRoleOutput.Role
	}

	// Get role tags
	listRoleTagsOutput, err := comp.iamAPI.ListRoleTags(&iam.ListRoleTagsInput{RoleName: role.RoleName})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_role: failed to list role tags")
	}

	var foundOperatorTag, foundKiamTag bool
	for _, tags := range listRoleTagsOutput.Tags {
		if aws.StringValue(tags.Key) == "ridecell-operator" {
			foundOperatorTag = true
		}
		if aws.StringValue(tags.Key) == "Kiam" {
			foundKiamTag = true
		}
	}
	if !foundOperatorTag || !foundKiamTag {
		_, err = comp.iamAPI.TagRole(&iam.TagRoleInput{
			RoleName: role.RoleName,
			Tags: []*iam.Tag{
				&iam.Tag{
					Key:   aws.String("ridecell-operator"),
					Value: aws.String("True"),
				},
				&iam.Tag{
					Key:   aws.String("Kiam"),
					Value: aws.String("true"),
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: failed to tag role")
		}
	}

	// Get inline role policy names
	listRolePoliciesOutput, err := comp.iamAPI.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: role.RoleName})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_role: failed to list inline role policies")
	}

	rolePolicies := map[string]string{}
	for _, rolePolicyName := range listRolePoliciesOutput.PolicyNames {
		getRolePolicy, err := comp.iamAPI.GetRolePolicy(&iam.GetRolePolicyInput{
			PolicyName: rolePolicyName,
			RoleName:   role.RoleName,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: failed to get role policy %s", aws.StringValue(rolePolicyName))
		}
		// No really, PolicyDocument is URL-encoded. I have no idea why. https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetRolePolicy.html
		decoded, err := url.PathUnescape(aws.StringValue(getRolePolicy.PolicyDocument))
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: error URL-decoding existing role policy %s", aws.StringValue(rolePolicyName))
		}
		rolePolicies[aws.StringValue(getRolePolicy.PolicyName)] = decoded
	}

	// If there is an inline policy that is not in the spec delete it
	for rolePolicyName := range rolePolicies {
		_, ok := inlinePolicies[rolePolicyName]
		if !ok {
			_, err = comp.iamAPI.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
				PolicyName: aws.String(rolePolicyName),
				RoleName:   role.RoleName,
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_role: failed to delete role policy %s", rolePolicyName)
			}
		}
	}

	// Update our role policies
	for policyName, policyJSON := range inlinePolicies {
		// Check for malformed JSON before we even try sending it.
		var specPolicyObj interface{}
		err := json.Unmarshal([]byte(policyJSON), &specPolicyObj)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: role policy from spec %s has invalid JSON", policyName)
		}

		// If a policy with the same name was returned compare it to our spec
		existingPolicy, ok := rolePolicies[policyName]
		if ok {
			var existingPolicyObj interface{}
			// Compare current policy to policy in spec
			err = json.Unmarshal([]byte(existingPolicy), &existingPolicyObj)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_role: existing role policy %s has invalid JSON (%v)", policyName, existingPolicy)
			}
			if reflect.DeepEqual(existingPolicyObj, specPolicyObj) {
				continue
			}
		}

		_, err = comp.iamAPI.PutRolePolicy(&iam.PutRolePolicyInput{
			PolicyDocument: aws.String(policyJSON),
			PolicyName:     aws.String(policyName),
			RoleName:       role.RoleName,
		})
		if err != nil {
			glog.Errorf("[%s/%s] iamrole: error putting role policy: %#v %#v %#v", instance.Namespace, instance.Name, *role.RoleName, policyName, policyJSON)
			return components.Result{}, errors.Wrapf(err, "iam_role: failed to put role policy %s", policyName)
		}
	}

	// Now we need to check the assumeRolePolicy
	// No really, PolicyDocument is URL-encoded. I have no idea why. https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetRole.html
	decodedExistingARPD, err := url.PathUnescape(aws.StringValue(role.AssumeRolePolicyDocument))
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_role: error URL-decoding existing assume role policy for %s", aws.StringValue(role.RoleName))
	}

	var exsitingARPDObj interface{}
	err = json.Unmarshal([]byte(decodedExistingARPD), &exsitingARPDObj)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_role: existing assume role policy for %s has invalid JSON", aws.StringValue(role.RoleName))
	}

	var specARPDObj interface{}
	err = json.Unmarshal([]byte(assumePolicyDocument), &specARPDObj)
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_role: assume role policy from spec for %s has invalid JSON", aws.StringValue(role.RoleName))
	}

	if !reflect.DeepEqual(exsitingARPDObj, specARPDObj) {
		_, err = comp.iamAPI.UpdateAssumeRolePolicy(&iam.UpdateAssumeRolePolicyInput{
			RoleName:       role.RoleName,
			PolicyDocument: aws.String(assumePolicyDocument),
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: unable to update assume role policy for %s", aws.StringValue(role.RoleName))
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*awsv1beta1.IAMRole)
		instance.Status.Status = awsv1beta1.StatusReady
		instance.Status.Message = "Role exists"
		return nil
	}}, nil
}

func (comp *iamRoleComponent) deleteDependencies(roleName string) (components.Result, error) {
	// check if the role exists before listing policies to prevent AccessDenied IAM edge case
	_, err := comp.iamAPI.GetRole(&iam.GetRoleInput{RoleName: aws.String(roleName)})
	if err != nil {
		aerr, ok := err.(awserr.Error)
		if ok && aerr.Code() == iam.ErrCodeNoSuchEntityException {
			// role no longer exists, can exit early
			return components.Result{}, nil
		}
		return components.Result{}, errors.Wrapf(aerr, "iam_role: failed to get role")
	}
	// Have to delete attached policies before role deletion
	listRolePoliciesOutput, err := comp.iamAPI.ListRolePolicies(&iam.ListRolePoliciesInput{RoleName: aws.String(roleName)})
	// If the role doesn't exist skip error
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(err, "iam_role: failed to list role policies for finalizer")
		}
	}
	for _, rolePolicy := range listRolePoliciesOutput.PolicyNames {
		_, err = comp.iamAPI.DeleteRolePolicy(&iam.DeleteRolePolicyInput{
			RoleName:   aws.String(roleName),
			PolicyName: rolePolicy,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_role: failed to delete role policy for finalizer")
		}
	}
	_, err = comp.iamAPI.DeleteRole(&iam.DeleteRoleInput{RoleName: aws.String(roleName)})
	// If the role doesn't exist skip error
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(aerr, "iam_role: failed to delete role for finalizer")
		}
	}
	return components.Result{}, nil
}

func (td *templatingData) parseField(field string) (string, error) {
	buff := &bytes.Buffer{}

	// Create template
	nameTemplate, err := template.New("").Parse(field)
	if err != nil {
		return "", errors.Wrapf(err, "iam_role: could not parse template")
	}

	// Swap template delimiters to [[]] and execute
	err = nameTemplate.Delims("[[", "]]").Execute(buff, td)
	if err != nil {
		return "", errors.Wrapf(err, "iam_role: could not execute template")
	}
	return buff.String(), nil
}
