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

package encryptedsecret_test

import (
	"os"
	"time"

	"github.com/Ridecell/ridecell-operator/pkg/test_helpers"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/sts"

	secretsv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/secrets/v1beta1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("encryptedsecret controller", func() {
	var helpers *test_helpers.PerTestHelpers
	var encryptedSecret *secretsv1beta1.EncryptedSecret

	timeout := 10 * time.Second

	BeforeEach(func() {
		helpers = testHelpers.SetupTest()
		if os.Getenv("AWS_TESTING_ACCOUNT_ID") == "" {
			Skip("$AWS_TESTING_ACCOUNT_ID not set, skipping encryptedsecret integration tests")
		}

		sess, err := session.NewSession()
		Expect(err).NotTo(HaveOccurred())

		// Check if this being run on the testing account
		stssvc := sts.New(sess)
		getCallerIdentityOutput, err := stssvc.GetCallerIdentity(&sts.GetCallerIdentityInput{})
		Expect(err).NotTo(HaveOccurred())
		if aws.StringValue(getCallerIdentityOutput.Account) != os.Getenv("AWS_TESTING_ACCOUNT_ID") {
			Skip("These tests should only be run on the testing account.")
		}

		encryptedSecret = &secretsv1beta1.EncryptedSecret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test",
				Namespace: helpers.Namespace,
			},
		}
	})

	AfterEach(func() {
		helpers.TeardownTest()
	})

	It("decrypts all the things", func() {
		c := helpers.TestClient
		encryptedSecret.Data = map[string]string{
			"test0": "AQICAHgUEQrO6I+fa8ZPkYjL7Abmd3c5ORtX48tgF/JG4rV4HwERQgY81p0n5GOxdB/lpPvyAAAAaTBnBgkqhkiG9w0BBwagWjBYAgEAMFMGCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQMa3XV8GbVwxDhsbtbAgEQgCbZbEvLzXFSadWkPiYKcd9cqg8q7yjkckZLr5EPw/uYGcd8x33gVA==",
			"test1": "AQICAHgUEQrO6I+fa8ZPkYjL7Abmd3c5ORtX48tgF/JG4rV4HwEI4o0ZLlI4tL22DQrpiYaiAAAAczBxBgkqhkiG9w0BBwagZDBiAgEAMF0GCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQME0AnQC5DkGDJtY/cAgEQgDAKnBDvfjm3Lmnt9/fJusAqZyToGC8U0nFI4YJ7NPQ4b6iiykFTeUpcSzvL6xFz5iE=",
			"test2": "AQICAHgUEQrO6I+fa8ZPkYjL7Abmd3c5ORtX48tgF/JG4rV4HwHz6BaXpwyS8KX6O9+vowTDAAAAbDBqBgkqhkiG9w0BBwagXTBbAgEAMFYGCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQMWb/IZcZww1Mkji12AgEQgCmZhPFQ+MDam/TUz0oEbQP3qQykP/BtxHxMno/HxMhClm/gHCrEmk+r8A==",
			"test3": "AQICAHgUEQrO6I+fa8ZPkYjL7Abmd3c5ORtX48tgF/JG4rV4HwHKNgrqBEXbBzEwAuMHVkhHAAAAfTB7BgkqhkiG9w0BBwagbjBsAgEAMGcGCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQMuIrBaPfAoWjetMyZAgEQgDpYg5mJro4tG8SHe9w8ipibGfqiQdhkXaP3wRGJr35ibYXA+NKO8h6SSJYyvSis+qMPzluZC92ul3fN",
		}
		c.Create(encryptedSecret)

		fetchSecret := &corev1.Secret{}
		c.EventuallyGet(helpers.Name("test"), fetchSecret, c.EventuallyTimeout(timeout))

		Expect(string(fetchSecret.Data["test0"])).To(Equal("Testing1234"))
		Expect(string(fetchSecret.Data["test1"])).To(Equal("thisisanecryptedvalue"))
		Expect(string(fetchSecret.Data["test2"])).To(Equal("moretestvalues"))
		Expect(string(fetchSecret.Data["test3"])).To(Equal(`\i\heard\you\like\back\slashes\`))

		fetchEncryptedSecret := &secretsv1beta1.EncryptedSecret{}
		c.EventuallyGet(helpers.Name("test"), fetchEncryptedSecret, c.EventuallyStatus(secretsv1beta1.StatusReady))
	})

	It("tries to decrypt key with wrong or missing encryption context", func() {
		c := helpers.TestClient
		encryptedSecret.Data = map[string]string{
			"test0": "AQICAHgUEQrO6I+fa8ZPkYjL7Abmd3c5ORtX48tgF/JG4rV4HwHTV2KY7GESZLiaoCMcPjzPAAAAajBoBgkqhkiG9w0BBwagWzBZAgEAMFQGCSqGSIb3DQEHATAeBglghkgBZQMEAS4wEQQMDmRumsVLTzZyNfAYAgEQgCdPxW0mgkd1DD3/iJZ+CV2yrQx9oz09oaAFQNO5cZzu875/q/PxHBY=",
		}
		c.Create(encryptedSecret)

		fetchEncryptedSecret := &secretsv1beta1.EncryptedSecret{}
		c.EventuallyGet(helpers.Name("test"), fetchEncryptedSecret, c.EventuallyStatus(secretsv1beta1.StatusError))
	})

	It("tries to decrypt invalid ciphertext", func() {
		c := helpers.TestClient
		encryptedSecret.Data = map[string]string{
			"test0": "invalid",
		}
		c.Create(encryptedSecret)

		fetchEncryptedSecret := &secretsv1beta1.EncryptedSecret{}
		c.EventuallyGet(helpers.Name("test"), fetchEncryptedSecret, c.EventuallyStatus(secretsv1beta1.StatusError))
	})

	It("tries to decrypt a key without a value", func() {
		c := helpers.TestClient
		encryptedSecret.Data = map[string]string{
			"test0": "",
		}
		c.Create(encryptedSecret)

		fetchEncryptedSecret := &secretsv1beta1.EncryptedSecret{}
		c.EventuallyGet(helpers.Name("test"), fetchEncryptedSecret, c.EventuallyStatus(secretsv1beta1.StatusError))
	})
})
