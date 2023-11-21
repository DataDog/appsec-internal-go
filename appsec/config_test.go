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
	t.Run("API Security", func(t *testing.T) {
		for _, tc := range []struct {
			name          string
			enabledVar    string
			sampleRateVar string
			enabled       bool
			sampleRate    float64
		}{
			{
				name:       "disabled",
				sampleRate: defaultAPISecSampleRate,
			},
			{
				name:       "disabled",
				enabledVar: "false",
				sampleRate: defaultAPISecSampleRate,
			},
			{
				name:       "disabled",
				enabledVar: "0",
				sampleRate: defaultAPISecSampleRate,
			},
			{
				name:       "disabled",
				enabledVar: "weirdvalue",
				sampleRate: defaultAPISecSampleRate,
			},
			{
				name:       "enabled",
				enabledVar: "true",
				enabled:    true,
				sampleRate: defaultAPISecSampleRate,
			},
			{
				name:       "enabled",
				enabledVar: "1",
				enabled:    true,
				sampleRate: defaultAPISecSampleRate,
			},
			{
				name:          "sampleRate 1.0",
				enabledVar:    "true",
				sampleRateVar: "1.0",
				enabled:       true,
				sampleRate:    1.0,
			},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Setenv(envAPISecEnabled, tc.enabledVar)
				t.Setenv(envAPISecSampleRate, tc.sampleRateVar)
				cfg := NewAPISecConfig()
				require.Equal(t, tc.enabled, cfg.Enabled)
				require.Equal(t, tc.sampleRate, cfg.SampleRate)
			})
		}

	})
}

func TestObfuscatorConfig(t *testing.T) {
	defaultConfig := ObfuscatorConfig{
		KeyRegex:   defaultObfuscatorKeyRegex,
		ValueRegex: defaultObfuscatorValueRegex,
	}
	t.Run("obfuscator", func(t *testing.T) {
		t.Run("key-regexp", func(t *testing.T) {
			t.Run("env-var-normal", func(t *testing.T) {
				expCfg := defaultConfig
				expCfg.KeyRegex = "test"
				t.Setenv(envObfuscatorKey, "test")
				cfg := NewObfuscatorConfig()
				require.Equal(t, expCfg, cfg)
			})
			t.Run("env-var-empty", func(t *testing.T) {
				expCfg := defaultConfig
				expCfg.KeyRegex = ""
				t.Setenv(envObfuscatorKey, "")
				cfg := NewObfuscatorConfig()
				require.Equal(t, expCfg, cfg)
			})
			t.Run("compile-error", func(t *testing.T) {
				t.Setenv(envObfuscatorKey, "+")
				cfg := NewObfuscatorConfig()
				require.Equal(t, defaultConfig, cfg)
			})
		})

		t.Run("value-regexp", func(t *testing.T) {
			t.Run("env-var-normal", func(t *testing.T) {
				expCfg := defaultConfig
				expCfg.ValueRegex = "test"
				t.Setenv(envObfuscatorValue, "test")
				cfg := NewObfuscatorConfig()
				require.Equal(t, expCfg, cfg)
			})
			t.Run("env-var-empty", func(t *testing.T) {
				expCfg := defaultConfig
				expCfg.ValueRegex = ""
				t.Setenv(envObfuscatorValue, "")
				cfg := NewObfuscatorConfig()
				require.Equal(t, expCfg, cfg)
			})
			t.Run("compile-error", func(t *testing.T) {
				t.Setenv(envObfuscatorValue, "+")
				cfg := NewObfuscatorConfig()
				require.Equal(t, defaultConfig, cfg)
			})
		})
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
			expected: defaultTraceRate,
		},
		{
			name:     "negative",
			env:      "-1",
			expected: defaultTraceRate,
		},
		{
			name:     "zero",
			env:      "0",
			expected: defaultTraceRate,
		},
		{
			name:     "empty-string",
			env:      "",
			expected: defaultTraceRate,
		},
	} {
		t.Run("trace-rate-limit/"+tc.name, func(t *testing.T) {
			t.Setenv(envTraceRateLimit, tc.env)
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
			expected: defaultWAFTimeout,
		},
		{
			name:     "negative",
			env:      "-1s",
			expected: defaultWAFTimeout,
		},
		{
			name:     "zero",
			env:      "0",
			expected: defaultWAFTimeout,
		},
		{
			name:     "empty-string",
			env:      "",
			expected: defaultWAFTimeout,
		},
	} {
		t.Run("waf-timeout/"+tc.name, func(t *testing.T) {
			t.Setenv(envWafTimeout, tc.env)
			require.Equal(t, tc.expected, WAFTimeoutFromEnv())
		})

	}
}

func TestRules(t *testing.T) {
	t.Run("rules", func(t *testing.T) {
		t.Run("empty-string", func(t *testing.T) {
			t.Setenv(envRules, "")
			rules, err := RulesFromEnv()
			require.NoError(t, err)
			require.Equal(t, StaticRecommendedRules, string(rules))
		})

		t.Run("file-not-found", func(t *testing.T) {
			t.Setenv(envRules, "i do not exist")
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
			t.Setenv(envRules, file.Name())
			rules, err := RulesFromEnv()
			require.NoError(t, err)
			require.Equal(t, StaticRecommendedRules, string(rules))
		})
	})
}
