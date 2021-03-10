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
	"bytes"
	"encoding/base64"
	"encoding/gob"
	"strings"

	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/kms"
	"github.com/aws/aws-sdk-go/service/kms/kmsiface"
	"github.com/pkg/errors"
	"golang.org/x/crypto/nacl/secretbox"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type payload struct {
	Key     []byte
	Nonce   *[24]byte
	Message []byte
}

type EncryptedSecretComponent struct {
	kmsAPI kmsiface.KMSAPI
}

func (comp *EncryptedSecretComponent) InjectKMSAPI(kmsapi kmsiface.KMSAPI) {
	comp.kmsAPI = kmsapi
}

func NewEncryptedSecret() *EncryptedSecretComponent {
	sess := session.Must(session.NewSessionWithOptions(session.Options{
		SharedConfigState: session.SharedConfigEnable,
		Config: aws.Config{
			Region: aws.String("us-west-1"),
		},
	}))
	kmsService := kms.New(sess)
	return &EncryptedSecretComponent{kmsAPI: kmsService}
}

func (_ *EncryptedSecretComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{
		&corev1.Secret{},
	}
}

func (_ *EncryptedSecretComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *EncryptedSecretComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*secretsv1beta1.EncryptedSecret)

	newSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:      instance.Name,
			Namespace: instance.Namespace,
		},
		Type: corev1.SecretTypeOpaque,
		Data: make(map[string][]byte),
	}

	// Key map for holding plainDataKey to avoid repetative KMS decrypt calls for single cipherDataKey
	keyMap := map[string]*[32]byte{}

	for k, v := range instance.Data {
		if v == "" {
			return components.Result{}, errors.Errorf("encryptedsecret: secret[%s] does not have a value", k)
		}
		useDataKey := false
		// check if value has crypto prefix, if true, then decrypt using data key
		if strings.HasPrefix(v, "crypto") {
			useDataKey = true
			array := strings.Split(v, " ")
			v = array[len(array)-1]
		}

		decodedValue, err := base64.StdEncoding.DecodeString(v)
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "encryptedsecret: failed to base64 decode secret")
		}

		if useDataKey {
			var p payload
			err = gob.NewDecoder(bytes.NewReader(decodedValue)).Decode(&p)
			if err != nil {
				return components.Result{}, errors.Wrapf(err, "error decoding value for payload")
			}
			plainDataKey, ok := keyMap[string(p.Key)]
			if !ok {
				// Decrypt cipherdatakey
				decryptedValue, err := comp.kmsAPI.Decrypt(&kms.DecryptInput{
					CiphertextBlob: p.Key,
					EncryptionContext: map[string]*string{
						"RidecellOperator": aws.String("true"),
					},
				})
				if err != nil {
					return components.Result{}, errors.Wrapf(err, "error decrypting value for cipherDatakey")
				}
				plainDataKey = &[32]byte{}
				copy(plainDataKey[:], decryptedValue.Plaintext)
				keyMap[string(p.Key)] = plainDataKey
			}
			// Decrypt message
			var plaintext []byte
			plaintext, ok = secretbox.Open(plaintext, p.Message, p.Nonce, plainDataKey)
			if !ok {
				return components.Result{}, errors.Wrapf(err, "error decrypting value with data key for %s", k)
			}
			if bytes.Equal(plaintext, []byte(secretsv1beta1.EncryptedSecretEmptyKey)) {
				// Decode the magic value to an empty string.
				newSecret.Data[k] = []byte{}
			} else {
				newSecret.Data[k] = plaintext
			}
			continue
		}

		decryptedValue, err := comp.kmsAPI.Decrypt(&kms.DecryptInput{
			CiphertextBlob: decodedValue,
			EncryptionContext: map[string]*string{
				"RidecellOperator": aws.String("true"),
			},
		})
		if err != nil {
			return components.Result{}, errors.Wrapf(err, "encryptedsecret: failed to decrypt secret")
		}
		if bytes.Equal(decryptedValue.Plaintext, []byte(secretsv1beta1.EncryptedSecretEmptyKey)) {
			// Decode the magic value to an empty string.
			newSecret.Data[k] = []byte{}
		} else {
			newSecret.Data[k] = decryptedValue.Plaintext
		}
	}

	_, err := controllerutil.CreateOrUpdate(ctx.Context, ctx, newSecret.DeepCopy(), func(existingObj runtime.Object) error {
		existing := existingObj.(*corev1.Secret)
		// Sync important fields.
		err := controllerutil.SetControllerReference(instance, existing, ctx.Scheme)
		if err != nil {
			return errors.Wrapf(err, "encryptedsecret: Failed to set controller reference")
		}
		existing.Labels = newSecret.Labels
		existing.Annotations = newSecret.Annotations
		existing.Type = newSecret.Type
		existing.Data = newSecret.Data
		return nil
	})
	if err != nil {
		return components.Result{}, errors.Wrapf(err, "encryptedsecret: failed to create or update secret")
	}

	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*secretsv1beta1.EncryptedSecret)
		instance.Status.Status = secretsv1beta1.StatusReady
		instance.Status.Message = "Secret Created"
		return nil
	}}, nil
}
