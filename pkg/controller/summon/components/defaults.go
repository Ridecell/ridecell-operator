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
	"fmt"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
)

const defaultFernetKeysLifespan = "8760h"

// Treat this as a const, no touchy.
var zeroSeconds time.Duration

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

func (comp *defaultsComponent) Reconcile(ctx *components.ComponentContext) (components.Result, error) {
	instance := ctx.Top.(*summonv1beta1.SummonPlatform)

	// Fill in defaults.
	if instance.Spec.Hostname == "" {
		instance.Spec.Hostname = instance.Name + ".ridecell.us"
	}
	if instance.Spec.Environment == "" {
		x := instance.Namespace
		if strings.HasPrefix(x, "summon-") {
			x = x[7:]
		}
		instance.Spec.Environment = x
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
	if len(instance.Spec.Secrets) == 0 {
		instance.Spec.Secrets = []string{instance.Namespace, instance.Name}
	}
	if instance.Spec.PullSecret == "" {
		instance.Spec.PullSecret = "pull-secret"
	}
	if instance.Spec.FernetKeyLifetime == zeroSeconds {
		// This is set to rotate fernet keys every year.
		parsedTimeDuration, _ := time.ParseDuration(defaultFernetKeysLifespan)
		instance.Spec.FernetKeyLifetime = parsedTimeDuration
	}
	if instance.Spec.AwsRegion == "" {
		instance.Spec.AwsRegion = "us-west-2"
	}
	if instance.Spec.SQSQueue == "" {
		if instance.Spec.Environment == "prod" || instance.Spec.Environment == "uat" {
			instance.Spec.SQSQueue = "prod-data-pipeline"
		} else {
			instance.Spec.SQSQueue = "master-data-pipeline"
		}
	}
	if instance.Spec.SQSRegion == "" {
		instance.Spec.SQSRegion = "us-west-2"
	}
	if instance.Spec.Database.SharedDatabaseName == "" {
		instance.Spec.Database.SharedDatabaseName = instance.Namespace
	}

	if instance.Spec.Environment == "uat" || instance.Spec.Environment == "prod" {
		defConfig("FIREBASE_APP", "ridecell")
	}

	if instance.Spec.EnableNewRelic == nil && instance.Spec.Environment == "prod" {
		val := true
		instance.Spec.EnableNewRelic = &val
	}

	if instance.Spec.Backup.TTL.Duration == 0 {
		instance.Spec.Backup.TTL.Duration = time.Hour * 720
		if instance.Spec.Environment == "dev" || instance.Spec.Environment == "qa" {
			instance.Spec.Backup.TTL.Duration = time.Hour * 72
		}
	}

	if instance.Spec.Backup.WaitUntilReady == nil {
		prodWaitBool := true
		instance.Spec.Backup.WaitUntilReady = &prodWaitBool
		if instance.Spec.Environment == "dev" || instance.Spec.Environment == "qa" {
			devWaitBool := false
			instance.Spec.Backup.WaitUntilReady = &devWaitBool
		}
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

	webURL := instance.Spec.Hostname
	if instance.Spec.Aliases != nil && len(instance.Spec.Aliases) > 0 {
		webURL = instance.Spec.Aliases[0]
	}
	defVal("WEB_URL", "https://%s", webURL)

	defVal("ASGI_URL", "redis://%s-redis/0", instance.Name)
	defVal("CACHE_URL", "redis://%s-redis/1", instance.Name)
	defVal("FIREBASE_ROOT_NODE", "%s", instance.Name)
	defVal("TENANT_ID", "%s", instance.Name)
	defVal("NEWRELIC_NAME", "%s-summon-platform", instance.Name)
	defVal("AWS_REGION", "%s", instance.Spec.AwsRegion)
	defVal("AWS_STORAGE_BUCKET_NAME", "ridecell-%s-static", instance.Name)
	defVal("DATA_PIPELINE_SQS_QUEUE_NAME", "%s", instance.Spec.SQSQueue)

	// Enable DEBUG automatically for dev/qa.
	if instance.Spec.Environment == "dev" || instance.Spec.Environment == "qa" {
		val := true
		instance.Spec.Config["DEBUG"] = summonv1beta1.ConfigValue{Bool: &val}
	}

	// Enable NewRelic if requested.
	if instance.Spec.EnableNewRelic != nil && *instance.Spec.EnableNewRelic {
		val := true
		instance.Spec.Config["ENABLE_NEW_RELIC"] = summonv1beta1.ConfigValue{Bool: &val}
	}

	return components.Result{}, nil
}

func defConfig(key string, value interface{}) {
	boolVal, ok := value.(bool)
	if ok {
		configDefaults[key] = summonv1beta1.ConfigValue{Bool: &boolVal}
		return
	}
	intVal, ok := value.(int32)
	if ok {
		configDefaults[key] = summonv1beta1.ConfigValue{Int: &intVal}
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
	defConfig("ALLOW_X_FORWARDED_PROTO", true)
	defConfig("AMAZON_S3_USED", true)
	defConfig("AMAZON_S3_MEDIA_ONLY", true)
	defConfig("AUTH_SDK_AUTH_SERVICE_PUBLIC_KEY", `-----BEGIN PUBLIC KEY-----
MIICIjANBgkqhkiG9w0BAQEFAAOCAg8AMIICCgKCAgEAsPk83VrFTv1yp8yY3j38
DlK93nZzu6QH3VoKe8VcbuEP7eixlKIt91ID67KCRQGYV/sWquTxP1bmBUrku7tx
nUXKs7NEchyMyhnq9/MaGenqv79QjpEzx1QikHplSPtp1Jj85ApWuECLgVfYuU1o
CkH5DFmyd7An5NCFjuU8On76KMbb05Mxmw0T10UVlftchP+aCAKuuqUFxcX6oVmw
kzWaFA13CCaeL2Vq1//ydYQtrhWEpx0fBsYq4nQsSz9wy37wbTcWVuyjMYG0Zyhh
Oer7gwhEQS+4Fbn5vluU0v4Fwy5Vo2sGJtYbsdMsQZIc11FJ6dRCOgS+oXcCouwS
a+KiQKrss4HuCovEwKxm3KgzaTOfgmVyf/39DUuBJ7cJuNe2mSJeRJvWSXBktTyS
gGXvsQ1JVRqbEAC0htjy4nKoCawvrIs1lO0CjfpxO5vEv4SPazGenHTGtN6RRIjk
PSQQAdjCUVnumveczncRwDkLmRWud7ijF74cqLgDAnUIeLJE3dqQv0Ff08R5Uh9b
WoyKbZrC1Ie5bd6OGix+GWOFtAZ6FQJ7fFVeOjCQkHOnYJfnorj0nlKTQXCWsDjq
waGIhRA2Oq1iha0fw8udSyUU+F0tWtaTAPrKe8VBWQPBwaWSzUjIP8Nb7EZBHLyP
ZSo/8E5P29isb34ZQedtc1kCAwEAAQ==
-----END PUBLIC KEY-----`)
	defConfig("CARSHARING_V1_API_DISABLED", false)
	defConfig("CLOUDFRONT_DISTRIBUTION", "")
	defConfig("CONN_MAX_AGE", int32(60))
	defConfig("COMPRESS_ENABLED", false)
	defConfig("CSBE_CONNECTION_USED", false)
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
	defConfig("REQUIRE_HTTPS", true)
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
	defConfig("SAML_VALID_FOR_HOURS", int32(24))
	defConfig("SESSION_COOKIE_AGE", int32(1209600))
	defConfig("TIME_ZONE", "America/Los_Angeles")
	defConfig("USE_FACEBOOK_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("USE_GOOGLE_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("USE_SAML_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("XMLSEC_BINARY_LOCATION", "/usr/bin/xmlsec1")
}
