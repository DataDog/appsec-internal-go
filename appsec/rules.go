// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2023-present Datadog, Inc.

package appsec

import "encoding/json"

// DefaultRuleset returns the marshaled default recommended security rules for AppSec
func DefaultRuleset() ([]byte, error) {
	return staticRecommendedRules, nil
}

// DefaultRulesetMap returns the unmarshaled default recommended security rules for AppSec
func DefaultRulesetMap() (map[string]any, error) {
	var rules map[string]any
	if err := json.Unmarshal(staticRecommendedRules, &rules); err != nil {
		return nil, err
	}

	return rules, nil
}
