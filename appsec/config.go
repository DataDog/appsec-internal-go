// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package appsec

import (
	"os"
	"regexp"
	"strconv"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/DataDog/datadog-agent/pkg/util/log"
)

// Configuration environment variables
const (
	envAPISecEnabled    = "DD_EXPERIMENTAL_API_SECURITY_ENABLED"
	envAPISecSampleRate = "DD_API_SECURITY_REQUEST_SAMPLE_RATE"
	envObfuscatorKey    = "DD_APPSEC_OBFUSCATION_PARAMETER_KEY_REGEXP"
	envObfuscatorValue  = "DD_APPSEC_OBFUSCATION_PARAMETER_VALUE_REGEXP"
	envWafTimeout       = "DD_APPSEC_WAF_TIMEOUT"
	envTraceRateLimit   = "DD_APPSEC_TRACE_RATE_LIMIT"
	envRules            = "DD_APPSEC_RULES"
)

// Configuration constants and default values
const (
	defaultAPISecSampleRate          = .1
	defaultObfuscatorKeyRegex        = `(?i)(?:p(?:ass)?w(?:or)?d|pass(?:_?phrase)?|secret|(?:api_?|private_?|public_?)key)|token|consumer_?(?:id|key|secret)|sign(?:ed|ature)|bearer|authorization`
	defaultObfuscatorValueRegex      = `(?i)(?:p(?:ass)?w(?:or)?d|pass(?:_?phrase)?|secret|(?:api_?|private_?|public_?|access_?|secret_?)key(?:_?id)?|token|consumer_?(?:id|key|secret)|sign(?:ed|ature)?|auth(?:entication|orization)?)(?:\s*=[^;]|"\s*:\s*"[^"]+")|bearer\s+[a-z0-9\._\-]+|token:[a-z0-9]{13}|gh[opsu]_[0-9a-zA-Z]{36}|ey[I-L][\w=-]+\.ey[I-L][\w=-]+(?:\.[\w.+\/=-]+)?|[\-]{5}BEGIN[a-z\s]+PRIVATE\sKEY[\-]{5}[^\-]+[\-]{5}END[a-z\s]+PRIVATE\sKEY|ssh-rsa\s*[a-z0-9\/\.+]{100,}`
	defaultWAFTimeout                = 4 * time.Millisecond
	defaultTraceRate            uint = 100 // up to 100 appsec traces/s
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
	value := os.Getenv(envAPISecSampleRate)
	rate, err := strconv.ParseFloat(value, 64)
	if err != nil {
		logEnvVarParsingError(envAPISecSampleRate, value, err, defaultAPISecSampleRate)
		return defaultAPISecSampleRate
	}
	// Clamp the value so that 0.0 <= rate <= 1.0
	if rate < 0. {
		rate = 0.
	} else if rate > 1. {
		rate = 1.
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
		logUnexpectedEnvVarValue(name, val, "could not compile the configured obfuscator regular expression", defaultValue)
		return defaultValue
	}
	log.Debug("appsec: starting with the configured obfuscator regular expression %s", name)
	return val
}

// WAFTimeoutFromEnv reads and parses the WAF timeout value set through the env
// If not set, it defaults to `defaultWAFTimeout`
func WAFTimeoutFromEnv() (timeout time.Duration) {
	timeout = defaultWAFTimeout
	value := os.Getenv(envWafTimeout)
	if value == "" {
		return
	}

	// Check if the value ends with a letter, which means the user has
	// specified their own time duration unit(s) such as 1s200ms.
	// Otherwise, default to microseconds.
	if lastRune, _ := utf8.DecodeLastRuneInString(value); !unicode.IsLetter(lastRune) {
		value += "us" // Add the default microsecond time-duration suffix
	}

	parsed, err := time.ParseDuration(value)
	if err != nil {
		logEnvVarParsingError(envWafTimeout, value, err, timeout)
		return
	}
	if parsed <= 0 {
		logUnexpectedEnvVarValue(envWafTimeout, parsed, "expecting a strictly positive duration", timeout)
		return
	}
	return parsed
}

// RateLimitFromEnv reads and parses the trace rate limit set through the env
// If not set, it defaults to `defaultTraceRate`
func RateLimitFromEnv() (rate uint) {
	rate = defaultTraceRate
	value := os.Getenv(envTraceRateLimit)
	if value == "" {
		return rate
	}
	parsed, err := strconv.ParseUint(value, 10, 0)
	if err != nil {
		logEnvVarParsingError(envTraceRateLimit, value, err, rate)
		return
	}
	if parsed == 0 {
		logUnexpectedEnvVarValue(envTraceRateLimit, parsed, "expecting a value strictly greater than 0", rate)
		return
	}
	return uint(parsed)
}

// RulesFromEnv returns the security rules provided through the environment
// If the env var is not set, the default recommended rules are returned instead
func RulesFromEnv() ([]byte, error) {
	filepath := os.Getenv(envRules)
	if filepath == "" {
		log.Debug("appsec: using the default built-in recommended security rules")
		return DefaultRuleset()
	}
	buf, err := os.ReadFile(filepath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Errorf("appsec: could not find the rules file in path %s: %v.", filepath, err)
		}
		return nil, err
	}
	log.Debug("appsec: using the security rules from file %s", filepath)
	return buf, nil
}

func logEnvVarParsingError(name, value string, err error, defaultValue any) {
	log.Errorf("appsec: could not parse the env var %s=%s as a duration: %v. Using default value %v.", name, value, err, defaultValue)
}

func logUnexpectedEnvVarValue(name string, value any, reason string, defaultValue any) {
	log.Errorf("appsec: unexpected configuration value of %s=%v: %s. Using default value %v.", name, value, reason, defaultValue)
}
