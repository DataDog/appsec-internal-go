// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package appsec

import (
	"os"
	"regexp"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Configuration environment variables
const (
	envAPISecEnabled    = "DD_EXPERIMENTAL_API_SECURITY_ENABLED"
	envAPISecSampleRate = "DD_API_SECURITY_REQUEST_SAMPLE_RATE"
	envObfuscatorKey    = "DD_APPSEC_OBFUSCATION_PARAMETER_KEY_REGEXP"
	envObfuscatorValue  = "DD_APPSEC_OBFUSCATION_PARAMETER_VALUE_REGEXP"
)

// Configuration constants and default values
const (
	defaultAPISecSampleRate     = 10. / 100
	defaultObfuscatorKeyRegex   = `(?i)(?:p(?:ass)?w(?:or)?d|pass(?:_?phrase)?|secret|(?:api_?|private_?|public_?)key)|token|consumer_?(?:id|key|secret)|sign(?:ed|ature)|bearer|authorization`
	defaultObfuscatorValueRegex = `(?i)(?:p(?:ass)?w(?:or)?d|pass(?:_?phrase)?|secret|(?:api_?|private_?|public_?|access_?|secret_?)key(?:_?id)?|token|consumer_?(?:id|key|secret)|sign(?:ed|ature)?|auth(?:entication|orization)?)(?:\s*=[^;]|"\s*:\s*"[^"]+")|bearer\s+[a-z0-9\._\-]+|token:[a-z0-9]{13}|gh[opsu]_[0-9a-zA-Z]{36}|ey[I-L][\w=-]+\.ey[I-L][\w=-]+(?:\.[\w.+\/=-]+)?|[\-]{5}BEGIN[a-z\s]+PRIVATE\sKEY[\-]{5}[^\-]+[\-]{5}END[a-z\s]+PRIVATE\sKEY|ssh-rsa\s*[a-z0-9\/\.+]{100,}`
)

// APISecConfig holds the configuration for API Security schemas reporting
// It is used to enabled/disable the feature as well as to configure the rate
// at which schemas get reported,
type APISecConfig struct {
	Enabled    bool
	SampleRate float64
}

// ObfuscatorConfig wraps the key and value regexp to be passed to the WAF to perform obfuscation.
type ObfuscatorConfig struct {
	KeyRegex   string
	ValueRegex string
}

// NewAPISecConfig creates and returns a new API Security configuration by reading the env
func NewAPISecConfig() APISecConfig {
	return APISecConfig{
		Enabled:    apiSecurityEnabled(),
		SampleRate: readAPISecuritySampleRate(),
	}
}

func apiSecurityEnabled() bool {
	enabled, _ := strconv.ParseBool(os.Getenv(envAPISecEnabled))
	return enabled
}

func readAPISecuritySampleRate() float64 {
	rate, err := strconv.ParseFloat(os.Getenv(envAPISecSampleRate), 64)
	if err != nil {
		log.Debugf("appsec: could not parse %s. Defaulting to %f", envAPISecSampleRate, defaultAPISecSampleRate)
		return defaultAPISecSampleRate
	}
	if rate < 0. || rate > 1. {
		log.Debugf("appsec: %s value must be between 0 and 1. Defaulting to %f", envAPISecSampleRate, defaultAPISecSampleRate)
		return defaultAPISecSampleRate
	}
	return rate
}

// NewObfuscatorConfig creates and returns a new WAF obfuscator configuration by reading the env
func NewObfuscatorConfig() ObfuscatorConfig {
	keyRE := readObfuscatorConfigRegexp(envObfuscatorKey, defaultObfuscatorKeyRegex)
	valueRE := readObfuscatorConfigRegexp(envObfuscatorValue, defaultObfuscatorValueRegex)
	return ObfuscatorConfig{KeyRegex: keyRE, ValueRegex: valueRE}
}

func readObfuscatorConfigRegexp(name, defaultValue string) string {
	val, present := os.LookupEnv(name)
	if !present {
		log.Debug("appsec: %s not defined, starting with the default obfuscator regular expression", name)
		return defaultValue
	}
	if _, err := regexp.Compile(val); err != nil {
		log.Errorf("appsec: could not compile the configured obfuscator regular expression `%s=%s`. Using the default value instead", name, val)
		return defaultValue
	}
	log.Debug("appsec: starting with the configured obfuscator regular expression %s", name)
	return val
}
