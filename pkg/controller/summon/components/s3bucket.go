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
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"

	awsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/aws/v1beta1"
	summonv1beta "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
)

type s3BucketComponent struct {
	templatePath string
	miv          bool
}

func NewS3Bucket(templatePath string) *s3BucketComponent {
	return &s3BucketComponent{templatePath: templatePath}
}

func NewMIVS3Bucket(templatePath string) *s3BucketComponent {
	comp := NewS3Bucket(templatePath)
	comp.miv = true
	return comp
}

func (comp *s3BucketComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&awsv1beta1.S3Bucket{},
	}
}

func (_ *s3BucketComponent) IsReconcilable(_ *components.ComponentContext) bool {
	// Has no dependencies, always reconcilable.
	return true
}

func (comp *s3BucketComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta.SummonPlatform)
	if comp.miv && instance.Spec.MIV.ExistingBucket != "" {
		// We are using an external bucket, make sure the operator-managed bucket is deleted if it exists.
		obj, err := ctx.GetTemplate(comp.templatePath, nil)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "s3bucket: error rendering template %s", comp.templatePath)
		}
		bucket := obj.(*awsv1beta1.S3Bucket)
		err = ctx.Delete(ctx.Context, obj)
		if err != nil && !kerrors.IsNotFound(err) {
			return components.Result{}, errors.Wrapf(err, "s3bucket: error deleting existing bucket %s/%s", bucket.Namespace, bucket.Name)
		}
		return components.Result{StatusModifier: func(obj runtime.Object) error {
			instance := obj.(*summonv1beta.SummonPlatform)
			instance.Status.MIV.Bucket = instance.Spec.MIV.ExistingBucket
			return nil
		}}, nil
	}

	var goal *awsv1beta1.S3Bucket
	res, _, err := ctx.CreateOrUpdate(comp.templatePath, nil, func(goalObj, existingObj runtime.Object) error {
		goal = goalObj.(*awsv1beta1.S3Bucket)
		existing := existingObj.(*awsv1beta1.S3Bucket)
		// Temporary hack to respect current region in spec
		if existing.Spec.Region != goal.Spec.Region && existing.Spec.Region != "" {
			goal.Spec.Region = existing.Spec.Region
		}
		// Copy the Spec over.
		existing.Spec = goal.Spec
		return nil
	})
	if comp.miv {
		res.StatusModifier = func(obj runtime.Object) error {
			instance := obj.(*summonv1beta.SummonPlatform)
			instance.Status.MIV.Bucket = goal.Spec.BucketName
			return nil
		}
	}
	return res, err
}
