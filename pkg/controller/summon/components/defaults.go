/*
Copyright 2018 Ridecell, Inc.

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
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

var configDefaults map[string]summonv1beta1.ConfigValue

type defaultsComponent struct {
}

func NewDefaults() *defaultsComponent {
	return &defaultsComponent{}
}

func (_ *defaultsComponent) WatchTypes() []runtime.Object {
	return []runtime.Object{}
}

func (_ *defaultsComponent) IsReconcilable(_ *components.ComponentContext) bool {
	return true
}

func (comp *defaultsComponent) Reconcile(ctx *components.ComponentContext) (reconcile.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Fill in defaults.
	if instance.Spec.Hostname == "" {
		instance.Spec.Hostname = instance.Name + ".ridecell.us"
	}
	defaultReplicas := int32(1)
	if instance.Spec.WebReplicas == nil {
		instance.Spec.WebReplicas = &defaultReplicas
	}
	if instance.Spec.DaphneReplicas == nil {
		instance.Spec.DaphneReplicas = &defaultReplicas
	}
	if instance.Spec.WorkerReplicas == nil {
		instance.Spec.WorkerReplicas = &defaultReplicas
	}
	if instance.Spec.ChannelWorkerReplicas == nil {
		instance.Spec.ChannelWorkerReplicas = &defaultReplicas
	}
	if instance.Spec.StaticReplicas == nil {
		instance.Spec.StaticReplicas = &defaultReplicas
	}
	if instance.Spec.PullSecret == "" {
		instance.Spec.PullSecret = "pull-secret"
	}

	// Fernet key rotation default
	zeroSeconds, _ := time.ParseDuration("0s")
	if instance.Spec.FernetKeyRotation == zeroSeconds {
		// This is set to rotate fernet keys every year.
		parsedTimeDuration, _ := time.ParseDuration("8760h")
		instance.Spec.FernetKeyRotation = parsedTimeDuration
	}

	// Fill in static default config values.
	if instance.Spec.Config == nil {
		instance.Spec.Config = map[string]summonv1beta1.ConfigValue{}
	}
	for key, value := range configDefaults {
		_, ok := instance.Spec.Config[key]
		if !ok {
			instance.Spec.Config[key] = value
		}
	}

	// Fill in the two config values that need the instance name in them.
	defVal := func(key, valueTemplate string, args ...interface{}) {
		_, ok := instance.Spec.Config[key]
		if !ok {
			value := fmt.Sprintf(valueTemplate, args...)
			instance.Spec.Config[key] = summonv1beta1.ConfigValue{String: &value}
		}
	}
	defVal("ASGI_URL", "redis://%s-redis/0", instance.Name)
	defVal("CACHE_URL", "redis://%s-redis/1", instance.Name)
	defVal("FIREBASE_ROOT_NODE", "%s", instance.Name)
	defVal("TENANT_ID", "%s", instance.Name)
	defVal("WEB_URL", "https://%s", instance.Spec.Hostname)
	defVal("NEWRELIC_NAME", "%s-summon-platform", instance.Name)

	return reconcile.Result{}, nil
}

func defConfig(key string, value interface{}) {
	boolVal, ok := value.(bool)
	if ok {
		configDefaults[key] = summonv1beta1.ConfigValue{Bool: &boolVal}
		return
	}
	floatVal, ok := value.(float64)
	if ok {
		configDefaults[key] = summonv1beta1.ConfigValue{Float: &floatVal}
		return
	}
	stringVal, ok := value.(string)
	if ok {
		configDefaults[key] = summonv1beta1.ConfigValue{String: &stringVal}
		return
	}
	panic("Unknown type")
}

func init() {
	configDefaults = map[string]summonv1beta1.ConfigValue{}
	// Default config, mostly based on local dev.
	defConfig("AMAZON_S3_USED", false)
	defConfig("AWS_REGION", "us-west-2")
	defConfig("AWS_STORAGE_BUCKET_NAME", "")
	defConfig("CARSHARING_V1_API_DISABLED", false)
	defConfig("CLOUDFRONT_DISTRIBUTION", "")
	defConfig("COMPRESS_ENABLED", false)
	defConfig("CSBE_CONNECTION_USED", false)
	defConfig("DATA_PIPELINE_SQS_QUEUE_NAME", "master-data-pipeline")
	defConfig("DEBUG", false)
	defConfig("ENABLE_NEW_RELIC", false)
	defConfig("ENABLE_SENTRY", false)
	defConfig("FACEBOOK_AUTHENTICATION_EMPLOYEE_PERMISSION_REQUIRED", false)
	defConfig("FIREBASE_APP", "instant-stage")
	defConfig("GDPR_ENABLED", true)
	defConfig("GOOGLE_ANALYTICS_ID", "UA-37653074-1")
	defConfig("INTERNATIONAL_OUTGOING_SMS_NUMBER", "14152345773")
	defConfig("OAUTH_HOSTED_DOMAIN", "")
	defConfig("OUTGOING_SMS_NUMBER", "41254")
	defConfig("PLATFORM_ENV", "DEV")
	defConfig("SAML_EMAIL_ATTRIBUTE", "eduPersonPrincipalName")
	defConfig("SAML_FIRST_NAME_ATTRIBUTE", "givenName")
	defConfig("SAML_IDP_ENTITY_ID", "https://idp.testshib.org/idp/shibboleth")
	defConfig("SAML_IDP_METADATA_FILENAME", "")
	defConfig("SAML_IDP_METADATA_URL", "https://www.testshib.org/metadata/testshib-providers.xml")
	defConfig("SAML_IDP_PUBLIC_KEY_FILENAME", "testshib.crt")
	defConfig("SAML_IDP_SSO_URL", "https://idp.testshib.org/idp/profile/SAML2/Redirect/SSO")
	defConfig("SAML_LAST_NAME_ATTRIBUTE", "sn")
	defConfig("SAML_NAME_ID_FORMAT", "urn:oasis:names:tc:SAML:2.0:nameid-format:transient")
	defConfig("SAML_PRIVATE_KEY_FILENAME", "sp.key")
	defConfig("SAML_PRIVATE_KEY_FILENAME", "sp.key")
	defConfig("SAML_PUBLIC_KEY_FILENAME", "sp.crt")
	defConfig("SAML_PUBLIC_KEY_FILENAME", "sp.crt")
	defConfig("SAML_SERVICE_NAME", "RideCell SAML Test")
	defConfig("SAML_USE_LOCAL_METADATA", "")
	defConfig("SAML_VALID_FOR_HOURS", float64(24))
	defConfig("SESSION_COOKIE_AGE", float64(1209600))
	defConfig("TIME_ZONE", "America/Los_Angeles")
	defConfig("USE_FACEBOOK_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("USE_GOOGLE_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("USE_SAML_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("XMLSEC_BINARY_LOCATION", "/usr/bin/xmlsec1")

}
