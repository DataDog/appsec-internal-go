// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package appsec

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestAPISecConfig(t *testing.T) {
	for _, tc := range []struct {
		name       string
		enabledVar string
		enabled    bool
	}{
		{
			name:       "disabled",
			enabledVar: "false",
		},
		{
			name:       "disabled",
			enabledVar: "0",
		},
		{
			name:    "enabled",
			enabled: true,
		},
		{
			name:       "enabled",
			enabledVar: "true",
			enabled:    true,
		},
		{
			name:       "enabled",
			enabledVar: "1",
			enabled:    true,
		},
		{
			name:       "enabled",
			enabledVar: "weirdvalue",
			enabled:    true,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if len(tc.enabledVar) > 0 {
				t.Setenv(EnvAPISecEnabled, tc.enabledVar)
			}
			cfg := NewAPISecConfig()
			require.Equal(t, tc.enabled, cfg.Enabled)
		})
	}
}

func TestObfuscatorConfig(t *testing.T) {
	defaultConfig := ObfuscatorConfig{
		KeyRegex:   DefaultObfuscatorKeyRegex,
		ValueRegex: DefaultObfuscatorValueRegex,
	}
	t.Run("key/env-var-normal", func(t *testing.T) {
		expCfg := defaultConfig
		expCfg.KeyRegex = "test"
		t.Setenv(EnvObfuscatorKey, "test")
		cfg := NewObfuscatorConfig()
		require.Equal(t, expCfg, cfg)
	})
	t.Run("key/env-var-empty", func(t *testing.T) {
		expCfg := defaultConfig
		expCfg.KeyRegex = ""
		t.Setenv(EnvObfuscatorKey, "")
		cfg := NewObfuscatorConfig()
		require.Equal(t, expCfg, cfg)
	})
	t.Run("key/compile-error", func(t *testing.T) {
		t.Setenv(EnvObfuscatorKey, "+")
		cfg := NewObfuscatorConfig()
		require.Equal(t, defaultConfig, cfg)
	})

	t.Run("value/env-var-normal", func(t *testing.T) {
		expCfg := defaultConfig
		expCfg.ValueRegex = "test"
		t.Setenv(EnvObfuscatorValue, "test")
		cfg := NewObfuscatorConfig()
		require.Equal(t, expCfg, cfg)
	})
	t.Run("value/env-var-empty", func(t *testing.T) {
		expCfg := defaultConfig
		expCfg.ValueRegex = ""
		t.Setenv(EnvObfuscatorValue, "")
		cfg := NewObfuscatorConfig()
		require.Equal(t, expCfg, cfg)
	})
	t.Run("value/compile-error", func(t *testing.T) {
		t.Setenv(EnvObfuscatorValue, "+")
		cfg := NewObfuscatorConfig()
		require.Equal(t, defaultConfig, cfg)
	})
}

func TestTraceRateLimit(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      string
		expected uint
	}{
		{
			name:     "parsable",
			env:      "1234567890",
			expected: 1234567890,
		},
		{
			name:     "not-parsable",
			env:      "not a uint",
			expected: DefaultTraceRate,
		},
		{
			name:     "negative",
			env:      "-1",
			expected: DefaultTraceRate,
		},
		{
			name:     "zero",
			env:      "0",
			expected: DefaultTraceRate,
		},
		{
			name:     "empty-string",
			env:      "",
			expected: DefaultTraceRate,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvTraceRateLimit, tc.env)
			require.Equal(t, tc.expected, RateLimitFromEnv())
		})
	}
}

func TestWAFTimeout(t *testing.T) {
	for _, tc := range []struct {
		name     string
		env      string
		expected time.Duration
	}{
		{
			name:     "parsable",
			env:      "5s",
			expected: 5 * time.Second,
		},
		{
			name:     "parsable-default-microsecond",
			env:      "1",
			expected: 1 * time.Microsecond,
		},
		{
			name:     "not-parsable",
			env:      "not a duration string",
			expected: DefaultWAFTimeout,
		},
		{
			name:     "negative",
			env:      "-1s",
			expected: DefaultWAFTimeout,
		},
		{
			name:     "zero",
			env:      "0",
			expected: DefaultWAFTimeout,
		},
		{
			name:     "empty-string",
			env:      "",
			expected: DefaultWAFTimeout,
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(EnvWAFTimeout, tc.env)
			require.Equal(t, tc.expected, WAFTimeoutFromEnv())
		})

	}
}

func TestRules(t *testing.T) {
	t.Run("empty-string", func(t *testing.T) {
		defaultRules, err := DefaultRuleset()
		require.NoError(t, err)
		t.Setenv(EnvRules, "")
		rules, err := RulesFromEnv()
		require.NoError(t, err)
		require.Equal(t, defaultRules, rules)
	})

	t.Run("file-not-found", func(t *testing.T) {
		t.Setenv(EnvRules, "i do not exist")
		rules, err := RulesFromEnv()
		require.Error(t, err)
		require.Nil(t, rules)
	})

	t.Run("local-file", func(t *testing.T) {
		file, err := os.CreateTemp("", "example-*")
		require.NoError(t, err)
		defer func() {
			file.Close()
			os.Remove(file.Name())
		}()
		_, err = file.WriteString(StaticRecommendedRules)
		require.NoError(t, err)
		t.Setenv(EnvRules, file.Name())
		rules, err := RulesFromEnv()
		require.NoError(t, err)
		require.Equal(t, StaticRecommendedRules, string(rules))
	})
}

func TestRASPEnablement(t *testing.T) {
	t.Run("enabled", func(t *testing.T) {
		t.Setenv(EnvRASPEnabled, "true")
		require.True(t, RASPEnabled())
	})

	t.Run("disabled", func(t *testing.T) {
		t.Setenv(EnvRASPEnabled, "false")
		require.False(t, RASPEnabled())
	})

	t.Run("unset", func(t *testing.T) {
		// t.Setenv first to restore the original value at the end of the test
		t.Setenv(EnvRASPEnabled, "")
		os.Unsetenv(EnvRASPEnabled)
		require.True(t, RASPEnabled())
	})
}

func TestDurationEnv(t *testing.T) {
	const varName = "DD_TEST_VARIABLE_DURATION"

	type testCase struct {
		EnvVal   string
		EnvUnit  string
		Expected time.Duration
	}
	testCases := map[string]testCase{
		"blank": {
			EnvVal:   "",
			EnvUnit:  "s",
			Expected: 1337,
		},
		"1m": {
			EnvVal:   "1",
			EnvUnit:  "m",
			Expected: time.Minute,
		},
		"invalid": {
			EnvVal:   "invalid",
			EnvUnit:  "s",
			Expected: 1337,
		},
	}

	for name, tc := range testCases {
		t.Run(name, func(t *testing.T) {
			t.Setenv(varName, tc.EnvVal)
			require.Equal(t, tc.Expected, durationEnv(varName, tc.EnvUnit, 1337))
		})
	}
}
