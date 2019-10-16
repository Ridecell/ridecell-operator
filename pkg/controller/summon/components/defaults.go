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
	"os"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/runtime"

	summonv1beta1 "github.com/Ridecell/ridecell-operator/pkg/apis/summon/v1beta1"
	"github.com/Ridecell/ridecell-operator/pkg/components"
	"github.com/pkg/errors"
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
	if instance.Spec.Environment == "" {
		x := instance.Namespace
		x = strings.TrimPrefix(x, "summon-")
		instance.Spec.Environment = x
	}
	if instance.Spec.Hostname == "" {
		baseHostname := ".ridecell.us"
		if instance.Spec.Environment == "uat" || instance.Spec.Environment == "prod" {
			baseHostname = ".ridecell.com"
		}
		instance.Spec.Hostname = instance.Name + baseHostname
	}
	err := comp.replicaDefaults(instance)
	if err != nil {
		return components.Result{}, errors.Wrap(err, "error setting replica defaults")
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
		instance.Spec.AwsRegion = os.Getenv("AWS_REGION")
		// If the env var isn't present, assume us-west-2. Mostly for local testing stuff.
		if instance.Spec.AwsRegion == "" {
			instance.Spec.AwsRegion = "us-west-2"
		}
	}
	if instance.Spec.SQSQueue == "" {
		switch instance.Spec.Environment {
		case "prod", "uat":
			switch instance.Spec.AwsRegion {
			case "eu-central-1":
				instance.Spec.SQSQueue = "eu-prod-data-pipeline"
			case "ap-south-1":
				instance.Spec.SQSQueue = "in-prod-data-pipeline"
			default:
				instance.Spec.SQSQueue = "prod-data-pipeline"
			}
		case "qa":
			instance.Spec.SQSQueue = "us-qa-data-pipeline"
		default:
			instance.Spec.SQSQueue = "master-data-pipeline"
		}
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

	if instance.Spec.MigrationOverrides.RedisHostname != "" {
		defVal("ASGI_URL", "redis://%s/1", instance.Spec.MigrationOverrides.RedisHostname)
		defVal("CACHE_URL", "redis://%s/1", instance.Spec.MigrationOverrides.RedisHostname)
	} else {
		defVal("ASGI_URL", "redis://%s-redis/0", instance.Name)
		defVal("CACHE_URL", "redis://%s-redis/1", instance.Name)
	}

	defVal("FIREBASE_ROOT_NODE", "%s", instance.Name)
	defVal("TENANT_ID", "%s", instance.Name)
	defVal("NEWRELIC_NAME", "%s-summon-platform", instance.Name)
	defVal("AWS_REGION", "%s", instance.Spec.AwsRegion)
	defVal("AWS_STORAGE_BUCKET_NAME", "ridecell-%s-static", instance.Name)
	defVal("DATA_PIPELINE_SQS_QUEUE_NAME", "%s", instance.Spec.SQSQueue)

	// Translate our aws region into a usable region
	untranslatedRegion := strings.Split(os.Getenv("AWS_REGION"), "-")[0]
	translatedRegion := untranslatedRegion
	if untranslatedRegion == "ap" {
		translatedRegion = "in"
	}

	// Set our gateway environment for GATEWAY_BASE_URL
	gatewayEnv := "prod"

	if instance.Spec.Environment == "uat" || instance.Spec.Environment == "prod" {
		defVal("FIREBASE_APP", "ridecell")
	}
	if instance.Spec.Environment == "dev" || instance.Spec.Environment == "qa" {
		// Enable DEBUG automatically for dev/qa.
		val := true
		instance.Spec.Config["DEBUG"] = summonv1beta1.ConfigValue{Bool: &val}

		gatewayEnv = "master"
	}
	// Use our translated region and gateway env to set GATEWAY_BASE_URL
	defVal("GATEWAY_BASE_URL", "https://global.%s.%s.svc.ridecell.io/", translatedRegion, gatewayEnv)

	// Enable NewRelic if requested.
	if instance.Spec.EnableNewRelic != nil && *instance.Spec.EnableNewRelic {
		val := true
		instance.Spec.Config["ENABLE_NEW_RELIC"] = summonv1beta1.ConfigValue{Bool: &val}
	}

	// Enable monitoring for prod env
	if instance.Spec.Environment == "prod" {
		instance.Spec.Monitoring.Enabled = true
	}

	return components.Result{}, nil
}

func (comp *defaultsComponent) replicaDefaults(instance *summonv1beta1.SummonPlatform) error {
	replicas := &instance.Spec.Replicas
	intp := func(i int32) *int32 { return &i }
	defaultsForEnv := func(dev, qa, uat, prod int32) *int32 {
		switch instance.Spec.Environment {
		case "prod":
			return intp(prod)
		case "uat":
			return intp(uat)
		case "qa":
			return intp(qa)
		default:
			return intp(dev)
		}
	}

	// Copy over the legacy values since those take priority over defaults. This is pulled out to make it easier to remove later.
	if replicas.Web == nil && instance.Spec.WebReplicas != nil {
		replicas.Web = instance.Spec.WebReplicas
	}
	if replicas.Daphne == nil && instance.Spec.DaphneReplicas != nil {
		replicas.Daphne = instance.Spec.DaphneReplicas
	}
	if replicas.ChannelWorker == nil && instance.Spec.ChannelWorkerReplicas != nil {
		replicas.ChannelWorker = instance.Spec.ChannelWorkerReplicas
	}
	if replicas.Celeryd == nil && instance.Spec.WorkerReplicas != nil {
		replicas.Celeryd = instance.Spec.WorkerReplicas
	}
	if replicas.Static == nil && instance.Spec.StaticReplicas != nil {
		replicas.Static = instance.Spec.StaticReplicas
	}
	if replicas.CeleryBeat == nil && instance.Spec.NoCelerybeat {
		replicas.CeleryBeat = intp(0)
	}

	// Fill in defaults based on environment.
	if replicas.Web == nil {
		replicas.Web = defaultsForEnv(1, 1, 2, 4)
	}
	if replicas.Celeryd == nil {
		replicas.Celeryd = defaultsForEnv(1, 1, 1, 4)
	}
	if replicas.Daphne == nil {
		replicas.Daphne = defaultsForEnv(1, 1, 2, 2)
	}
	if replicas.ChannelWorker == nil {
		replicas.ChannelWorker = defaultsForEnv(1, 1, 2, 4)
	}
	if replicas.Static == nil {
		replicas.Static = defaultsForEnv(1, 1, 2, 2)
	}
	if replicas.CeleryBeat == nil {
		replicas.CeleryBeat = intp(1)
	}

	// Quick error check.
	if !(*replicas.CeleryBeat == 0 || *replicas.CeleryBeat == 1) {
		return errors.Errorf("Invalid celerybeat replicas, must be exactly 0 or 1: %v", *replicas.CeleryBeat)
	}

	return nil
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
	defConfig("CONN_MAX_AGE", float64(60))
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
	defConfig("SAML_IDP_ENTITY_ID", "")
	defConfig("SAML_IDP_METADATA_FILENAME", "metadata.xml")
	defConfig("SAML_IDP_METADATA_URL", "")
	defConfig("SAML_IDP_PUBLIC_KEY_FILENAME", "idp.crt")
	defConfig("SAML_IDP_SSO_URL", "")
	defConfig("SAML_LAST_NAME_ATTRIBUTE", "sn")
	defConfig("SAML_NAME_ID_FORMAT", "urn:oasis:names:tc:SAML:2.0:nameid-format:transient")
	defConfig("SAML_PRIVATE_KEY_FILENAME", "sp.key")
	defConfig("SAML_PRIVATE_KEY_FILENAME", "sp.key")
	defConfig("SAML_PUBLIC_KEY_FILENAME", "sp.crt")
	defConfig("SAML_PUBLIC_KEY_FILENAME", "sp.crt")
	defConfig("SAML_SERVICE_NAME", "")
	defConfig("SAML_USE_LOCAL_METADATA", "")
	defConfig("SAML_VALID_FOR_HOURS", float64(24))
	defConfig("SESSION_COOKIE_AGE", float64(1209600))
	defConfig("TIME_ZONE", "America/Los_Angeles")
	defConfig("USE_FACEBOOK_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("USE_GOOGLE_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("USE_SAML_AUTHENTICATION_FOR_RIDERS", false)
	defConfig("XMLSEC_BINARY_LOCATION", "/usr/bin/xmlsec1")
	defConfig("POWERPACK_UUID", "a654e39b-8bd0-40d4-9bb2-03989890c235")
}
