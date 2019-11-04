/*
Copyright 2019 Ridecell, Inc.

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
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/awserr"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/iam"
	"github.com/aws/aws-sdk-go/service/iam/iamiface"
	"github.com/golang/glog"
	"github.com/pkg/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	helpers "github.com/Ridecell/ridecell-operator/pkg/apis/helpers"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

const iamUserFinalizer = "iamuser.finalizer"

type iamUserComponent struct {
	iamAPI iamiface.IAMAPI
}

func NewIAMUser() *iamUserComponent {
	sess := session.Must(session.NewSession())
	iamService := iam.New(sess)
	return &iamUserComponent{iamAPI: iamService}
}

func (comp *iamUserComponent) InjectIAMAPI(iamapi iamiface.IAMAPI) {
	comp.iamAPI = iamapi
}

func (_ *iamUserComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{&corev1.Secret{}}
}

func (_ *iamUserComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *iamUserComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.IAMUser)

	// if object is not being deleted
	if instance.ObjectMeta.DeletionTimestamp.IsZero() {
		// Is our finalizer attached to the object?
		if !helpers.ContainsFinalizer(iamUserFinalizer, instance) {
			instance.ObjectMeta.Finalizers = helpers.AppendFinalizer(iamUserFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iamuser: failed to update instance while adding finalizer")
			}
			return components.Result{Requeue: true}, nil
		}
	} else {
		if helpers.ContainsFinalizer(iamUserFinalizer, instance) {
			if !instance.Spec.SkipFinalizers {
				result, err := comp.deleteDependencies(ctx)
				if err != nil {
					return result, err
				}
			}
			// All operations complete, remove finalizer
			instance.ObjectMeta.Finalizers = helpers.RemoveFinalizer(iamUserFinalizer, instance)
			err := ctx.Update(ctx.Context, instance)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iamuser: failed to update instance while removing finalizer")
			}
			return components.Result{}, nil
		}
		// If object is being deleted and has no finalizer just exit.
		return components.Result{}, nil
	}

	// Try to get our user, if it can't be found create it
	var user *iam.User
	getUserOutput, err := comp.iamAPI.GetUser(&iam.GetUserInput{UserName: aws.String(instance.Spec.UserName)})
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(aerr, "iam_user: failed to get user")
		}
		// If user does not exist create it
		createUserOutput, err := comp.iamAPI.CreateUser(&iam.CreateUserInput{
			UserName:            aws.String(instance.Spec.UserName),
			PermissionsBoundary: aws.String(instance.Spec.PermissionsBoundaryArn),
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to create user")
		}
		user = createUserOutput.User
	} else {
		// If getUser did not return an error
		user = getUserOutput.User
	}

	// Get user tags

	listUserTagsOutput, err := comp.iamAPI.ListUserTags(&iam.ListUserTagsInput{UserName: user.UserName})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_user: failed to list user tags")
	}

	var foundTag bool
	for _, tags := range listUserTagsOutput.Tags {
		if aws.StringValue(tags.Key) == "ridecell-operator" {
			foundTag = true
		}
	}
	if !foundTag {
		_, err = comp.iamAPI.TagUser(&iam.TagUserInput{
			UserName: user.UserName,
			Tags: []*iam.Tag{
				&iam.Tag{
					Key:   aws.String("ridecell-operator"),
					Value: aws.String("True"),
				},
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to tag user")
		}
	}

	// Get inline user policy names
	listUserPoliciesOutput, err := comp.iamAPI.ListUserPolicies(&iam.ListUserPoliciesInput{UserName: user.UserName})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_user: failed to list inline user policies")
	}

	userPolicies := map[string]string{}
	for _, userPolicyName := range listUserPoliciesOutput.PolicyNames {
		// Not actually in use at the moment.
		getUserPolicy, err := comp.iamAPI.GetUserPolicy(&iam.GetUserPolicyInput{
			PolicyName: userPolicyName,
			UserName:   user.UserName,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to get user policy %s", aws.StringValue(userPolicyName))
		}
		// No really, PolicyDocument is URL-encoded. I have no idea why. https://docs.aws.amazon.com/IAM/latest/APIReference/API_GetUserPolicy.html
		decoded, err := url.PathUnescape(aws.StringValue(getUserPolicy.PolicyDocument))
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: error URL-decoding existing user policy %s", aws.StringValue(userPolicyName))
		}
		userPolicies[aws.StringValue(getUserPolicy.PolicyName)] = decoded
	}

	// If there is an inline policy that is not in the spec delete it
	for userPolicyName := range userPolicies {
		_, ok := instance.Spec.InlinePolicies[userPolicyName]
		if !ok {
			_, err = comp.iamAPI.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
				PolicyName: aws.String(userPolicyName),
				UserName:   user.UserName,
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_user: failed to delete user policy %s", userPolicyName)
			}
		}
	}

	// Update our user policies
	for policyName, policyJSON := range instance.Spec.InlinePolicies {
		// Check for malformed JSON before we even try sending it.
		var specPolicyObj interface{}
		err := json.Unmarshal([]byte(policyJSON), &specPolicyObj)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: user policy from spec %s has invalid JSON", policyName)
		}

		// If a policy with the same name was returned compare it to our spec
		existingPolicy, ok := userPolicies[policyName]
		if ok {
			var existingPolicyObj interface{}
			// Compare current policy to policy in spec
			err = json.Unmarshal([]byte(existingPolicy), &existingPolicyObj)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_user: existing user policy %s has invalid JSON (%v)", policyName, existingPolicy)
			}
			if reflect.DeepEqual(existingPolicyObj, specPolicyObj) {
				continue
			}
		}

		_, err = comp.iamAPI.PutUserPolicy(&iam.PutUserPolicyInput{
			PolicyDocument: aws.String(policyJSON),
			PolicyName:     aws.String(policyName),
			UserName:       user.UserName,
		})
		if err != nil {
			glog.Errorf("[%s/%s] iamuser: error putting user policy: %#v %#v %#v", instance.Namespace, instance.Name, *user.UserName, policyName, policyJSON)
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to put user policy %s", policyName)
		}
	}

	fetchAccessKey := &corev1.Secret{}
	err = ctx.Get(ctx.Context, types.NamespacedName{Name: fmt.Sprintf("%s.aws-credentials", instance.Name), Namespace: instance.Namespace}, fetchAccessKey)
	if err != nil {
		if !k8serrors.IsNotFound(err) {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to get aws-credentials secret")
		}
		fetchAccessKey = &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: fmt.Sprintf("%s.aws-credentials", instance.Name), Namespace: instance.Namespace}}
	}

	fetchAccessKeyID, ok0 := fetchAccessKey.Data["AWS_ACCESS_KEY_ID"]
	_, ok1 := fetchAccessKey.Data["AWS_SECRET_ACCESS_KEY"]

	existingAccessKeys, err := comp.iamAPI.ListAccessKeys(&iam.ListAccessKeysInput{UserName: user.UserName})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "iam_user: failed to list access keys")
	}

	if !ok0 || !ok1 {
		fetchAccessKeyID = []byte{}
	}

	var foundAccessKeyID bool
	for _, accessKeyMeta := range existingAccessKeys.AccessKeyMetadata {
		if aws.StringValue(accessKeyMeta.AccessKeyId) == string(fetchAccessKeyID) {
			foundAccessKeyID = true
		} else {
			// If the access key isn't known to the controller delete it
			_, err := comp.iamAPI.DeleteAccessKey(&iam.DeleteAccessKeyInput{
				AccessKeyId: accessKeyMeta.AccessKeyId,
				UserName:    user.UserName,
			})
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "iam_user: failed to delete access keys")
			}
		}
	}

	if !foundAccessKeyID {
		// Make new access key and put it in a secret
		createAccessKeyOutput, err := comp.iamAPI.CreateAccessKey(&iam.CreateAccessKeyInput{UserName: user.UserName})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to create new access key")
		}
		fetchAccessKey.Data = make(map[string][]byte)
		fetchAccessKey.Data["AWS_ACCESS_KEY_ID"] = []byte(aws.StringValue(createAccessKeyOutput.AccessKey.AccessKeyId))
		fetchAccessKey.Data["AWS_SECRET_ACCESS_KEY"] = []byte(aws.StringValue(createAccessKeyOutput.AccessKey.SecretAccessKey))

		_, err = controllerutil.CreateOrUpdate(ctx.Context, ctx, fetchAccessKey.DeepCopyObject(), func(existingObj runtime.Object) error {
			existing := existingObj.(*corev1.Secret)
			// Sync important fields.
			err := controllerutil.SetControllerReference(instance, existing, ctx.Scheme)
			if err != nil {
				return errors.Wrapf(err, "iam_user: Failed to set controller reference")
			}
			existing.Labels = fetchAccessKey.Labels
			existing.Annotations = fetchAccessKey.Annotations
			existing.Type = fetchAccessKey.Type
			existing.Data = fetchAccessKey.Data
			return nil
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iam_user: failed to create or update secret")
		}
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*awsv1beta1.IAMUser)
		instance.Status.Status = awsv1beta1.StatusReady
		instance.Status.Message = "User exists and has secret"
		return nil
	}}, nil
}

func (comp *iamUserComponent) deleteDependencies(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*awsv1beta1.IAMUser)
	// Have to delete access keys before user deletion
	listAccessKeysOutput, err := comp.iamAPI.ListAccessKeys(&iam.ListAccessKeysInput{UserName: aws.String(instance.Spec.UserName)})
	// If the user doesn't exist skip error
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(err, "iamuser: failed to list access keys for finalizer")
		}
	}
	for _, accessKey := range listAccessKeysOutput.AccessKeyMetadata {
		_, err = comp.iamAPI.DeleteAccessKey(&iam.DeleteAccessKeyInput{
			UserName:    aws.String(instance.Spec.UserName),
			AccessKeyId: accessKey.AccessKeyId,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iamuser: failed to delete access key for finalizer")
		}
	}
	// Have to delete attached policies before user deletion
	listUserPoliciesOutput, err := comp.iamAPI.ListUserPolicies(&iam.ListUserPoliciesInput{UserName: aws.String(instance.Spec.UserName)})
	// If the user doesn't exist skip error
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(err, "iamuser: failed to list user policies for finalizer")
		}
	}
	for _, userPolicy := range listUserPoliciesOutput.PolicyNames {
		_, err = comp.iamAPI.DeleteUserPolicy(&iam.DeleteUserPolicyInput{
			UserName:   aws.String(instance.Spec.UserName),
			PolicyName: userPolicy,
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "iamuser: failed to delete user policy for finalizer")
		}
	}
	_, err = comp.iamAPI.DeleteUser(&iam.DeleteUserInput{UserName: aws.String(instance.Spec.UserName)})
	// If the user doesn't exist skip error
	if err != nil {
		if aerr, ok := err.(awserr.Error); !ok || aerr.Code() != iam.ErrCodeNoSuchEntityException {
			return components.Result{}, errors.Wrapf(aerr, "iam_user: failed to delete user for finalizer")
		}
	}
	return components.Result{}, nil
}
