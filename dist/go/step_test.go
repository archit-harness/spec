// Code generated by scripts/generate.js; DO NOT EDIT.

// Copyright 2022 Harness, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package yaml

type StepTest struct {
	Envs      map[string]string      `json:"envs,omitempty"`
	Uses      string                 `json:"uses,omitempty"`
	With      map[string]interface{} `json:"with,omitempty"`
	Splitting *Splitting             `json:"splitting,omitempty"`
	Reports   []*Report              `json:"reports,omitempty"`
	Outputs   []string               `json:"outputs,omitempty"`
	Image     string                 `json:"image,omitempty"`
	Connector string                 `json:"connector,omitempty"`
	User      string                 `json:"user,omitempty"`
	Pull      string                 `json:"pull,omitempty"`
	Resources *Resources             `json:"resources,omitempty"`
}
