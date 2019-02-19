/*
Copyright 2018-2019 Ridecell, Inc.

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
	"crypto/sha256"
	"encoding/hex"
	"fmt"

	_ "github.com/lib/pq"
	"github.com/pkg/errors"
	"golang.org/x/crypto/bcrypt"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/Ridecell/ridecell-operator/pkg/components/postgres"
)

type databaseComponent struct{}

func NewDatabase() *databaseComponent {
	return &databaseComponent{}
}

func (_ *databaseComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *databaseComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *databaseComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.DjangoUser)
	secretName := fmt.Sprintf("%s.django-password", instance.Name)

	// Try to find the password to use.
	secret := &corev1.Secret{}
	err := ctx.Get(ctx.Context, types.NamespacedName{Name: secretName, Namespace: instance.Namespace}, secret)
	if err != nil {
		return components.Result{Requeue: true}, errors.Wrapf(err, "database: Unable to load password secret %s/%s", instance.Namespace, secretName)
	}
	password, ok := secret.Data["password"]
	if !ok {
		return components.Result{Requeue: true}, errors.Errorf("database: Password secret %s/%s has no key \"password\"", instance.Namespace, secretName)
	}
	hashedPassword, err := comp.hashPassword(password)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "database: Error hashing password")
	}

	// Connect to the database.
	db, err := postgres.Open(ctx, &instance.Spec.Database)
	if err != nil {
		return components.Result{Requeue: true}, err
	}

	// Big ass SQL.
	query := `
INSERT INTO auth_user (username, password, first_name, last_name, email, is_active, is_staff, is_superuser, date_joined)
  VALUES ($1, $2, $3, $4, $1, $5, $6, $7, NOW())
  ON CONFLICT (username) DO UPDATE SET
    password=EXCLUDED.password,
    first_name=EXCLUDED.first_name,
    last_name=EXCLUDED.last_name,
    email=EXCLUDED.email,
    is_active=EXCLUDED.is_active,
    is_staff=EXCLUDED.is_staff,
    is_superuser=EXCLUDED.is_superuser
  RETURNING id;`

	// Create the auth_user.
	row := db.QueryRow(query, instance.Spec.Email, hashedPassword, instance.Spec.FirstName, instance.Spec.LastName, instance.Spec.Active, instance.Spec.Staff, instance.Spec.Superuser)
	var id int
	err = row.Scan(&id)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "database: Error running auth_user query")
	}

	// Smaller ass SQL. The awkward SET field is because DO NOTHING doesn't work with RETURNING.
	query = `
INSERT INTO common_userprofile (user_id, is_jumio_verified, created_at, updated_at)
  VALUES ($1, false, NOW(), NOW())
  ON CONFLICT (user_id) DO UPDATE SET
    is_jumio_verified=common_userprofile.is_jumio_verified
  RETURNING id;`

	// Create the common_userprofile.
	row = db.QueryRow(query, id)
	var profileId int
	err = row.Scan(&profileId)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "database: Error running common_userprofile query")
	}

	// Medium ass SQL.
	query = `
INSERT INTO common_staff (user_profile_id, is_active, manager, dispatcher)
  VALUES ($1, $2, $3, $4)
  ON CONFLICT (user_profile_id) DO UPDATE SET
    is_active=EXCLUDED.is_active,
    manager=EXCLUDED.manager,
    dispatcher=EXCLUDED.dispatcher;
`

	// Create the common_staff.
	_, err = db.Exec(query, profileId, instance.Spec.Active, instance.Spec.Manager, instance.Spec.Dispatcher)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "database: Error running common_staff query")
	}

	// Success!
	return components.Result{StatusModifier: func(obj runtime.Object) error {
		instance := obj.(*summonv1beta1.DjangoUser)
		instance.Status.Status = summonv1beta1.StatusReady
		instance.Status.Message = fmt.Sprintf("User %v created", id)
		return nil
	}}, nil
}

func (comp *databaseComponent) hashPassword(password []byte) (string, error) {
	// Take the SHA256.
	digested := sha256.Sum256(password)

	// Hex encode it.
	encoded := make([]byte, hex.EncodedLen(len(digested)))
	hex.Encode(encoded, digested[:])

	// Bcrypt it.
	hashed, err := bcrypt.GenerateFromPassword(encoded, 12)
	if err != nil {
		return "", err
	}

	// Format like Django uses.
	return fmt.Sprintf("bcrypt_sha256$%s", hashed), nil
}
