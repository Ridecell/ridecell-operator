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

package migration_test

import ()

const timeout = time.Second * 30

var _ = Describe("Migration controller", func() {
	//var helpers *test_helpers.PerTestHelpers
	//
	//BeforeEach(func() {
	//	helpers = testHelpers.SetupTest()
	//	pullSecret := &corev1.Secret{ObjectMeta: metav1.ObjectMeta{Name: "pull-secret", Namespace: helpers.OperatorNamespace}, Type: "kubernetes.io/dockerconfigjson", StringData: map[string]string{".dockerconfigjson": "{\"auths\": {}}"}}
	//	err := helpers.Client.Create(context.TODO(), pullSecret)
	//	Expect(err).NotTo(HaveOccurred())
	//	appSecrets := &corev1.Secret{
	//		ObjectMeta: metav1.ObjectMeta{Name: "testsecret", Namespace: helpers.Namespace},
	//		Data: map[string][]byte{
	//			"filler": []byte{}}}
	//	err = helpers.Client.Create(context.TODO(), appSecrets)
	//	Expect(err).NotTo(HaveOccurred())
	//})
	//
	//AfterEach(func() {
	//	// Display some debugging info if the test failed.
	//	if CurrentGinkgoTestDescription().Failed {
	//		helpers.DebugList(&summonv1beta1.SummonPlatformList{})
	//	}
	//	helpers.TeardownTest()
	//})
})
