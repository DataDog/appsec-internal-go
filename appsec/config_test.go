// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package appsec

import (
	"testing"

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
