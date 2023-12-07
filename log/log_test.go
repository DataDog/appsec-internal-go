// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2022-present Datadog, Inc.

package log_test

import (
	"errors"
	"fmt"
	"math/rand"
	"testing"

	"github.com/DataDog/appsec-internal-go/log"
	"github.com/stretchr/testify/require"
)

func TestBackend(t *testing.T) {
	type logLevelName string
	const (
		TRACE    logLevelName = "Trace"
		DEBUG    logLevelName = "Debug"
		INFO     logLevelName = "Info"
		WARN     logLevelName = "Warn"
		ERROR    logLevelName = "Errorf"
		CRITICAL logLevelName = "Criticalf"
	)

	logInfo := map[logLevelName]*struct {
		called  bool
		message string
	}{
		TRACE:    {},
		DEBUG:    {},
		INFO:     {},
		WARN:     {},
		ERROR:    {},
		CRITICAL: {},
	}

	reset := func() {
		for _, status := range logInfo {
			status.called = false
			status.message = ""
		}
	}

	mockLogger := func(level logLevelName) func(string, ...any) {
		return func(format string, args ...any) {
			logInfo[level].called = true
			logInfo[level].message = fmt.Sprintf(format, args...)
		}
	}
	mockErrLogger := func(level logLevelName) func(string, ...any) error {
		return func(format string, args ...any) error {
			err := fmt.Errorf(format, args...)
			logInfo[level].called = true
			logInfo[level].message = err.Error()
			return err
		}
	}

	log.SetBackend(log.Backend{
		Trace:     mockLogger(TRACE),
		Debug:     mockLogger(DEBUG),
		Info:      mockLogger(INFO),
		Warn:      mockLogger(WARN),
		Errorf:    mockErrLogger(ERROR),
		Criticalf: mockErrLogger(CRITICAL),
	})

	for name, logger := range map[logLevelName]func(string, ...any){
		TRACE: log.Trace,
		DEBUG: log.Debug,
		INFO:  log.Info,
		WARN:  log.Warn,
	} {
		t.Run(string(name), func(t *testing.T) {
			defer reset()

			// Given
			randomInt := rand.Int()

			// When
			logger("%s %d", name, randomInt)

			// Then
			expectedMessage := fmt.Sprintf("%s %d", name, randomInt)
			for level, status := range logInfo {
				if level == name {
					require.True(t, status.called)
					require.Equal(t, expectedMessage, status.message)
					return
				}
				require.False(t, status.called)
			}
		})
	}

	for name, logger := range map[logLevelName]func(string, ...any) error{
		ERROR:    log.Errorf,
		CRITICAL: log.Criticalf,
	} {
		t.Run(string(name), func(t *testing.T) {
			defer reset()

			// Given
			cause := errors.New("cause")
			randomInt := rand.Int()

			// When
			err := logger("%s %d: %w", name, randomInt, cause)

			// Then
			expectedMessage := fmt.Sprintf("%s %d: %v", name, randomInt, cause)
			require.Equal(t, expectedMessage, err.Error())
			require.Equal(t, cause, errors.Unwrap(err))
			for level, status := range logInfo {
				if level == name {
					require.True(t, status.called)
					require.Equal(t, expectedMessage, status.message)
					return
				}
				require.False(t, status.called)
			}
		})
	}
}
