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

import (
	"encoding/json"
	"fmt"
)

type Stage struct {
	Desc     string        `json:"desc,omitempty"`
	Id       string        `json:"id,omitempty"`
	Name     string        `json:"name,omitempty"`
	Strategy *Strategy     `json:"strategy,omitempty"`
	Delegate Stringorslice `json:"delegate,omitempty"`
	Status   *StatusStage  `json:"status,omitempty"`
	Type     string        `json:"type,omitempty"`
	When     *When         `json:"when,omitempty"`
	On       *On           `json:"on,omitempty"`
	Spec     interface{}   `json:"spec,omitempty"`
}

// UnmarshalJSON implement the json.Unmarshaler interface.
func (v *Stage) UnmarshalJSON(data []byte) error {
	type S Stage
	type T struct {
		*S
		Spec json.RawMessage `json:"spec"`
	}

	obj := &T{S: (*S)(v)}
	if err := json.Unmarshal(data, obj); err != nil {
		return err
	}

	switch v.Type {
	case "ci":
		v.Spec = new(StageCI)
	case "cd":
		v.Spec = new(StageCD)
	case "custom":
		v.Spec = new(StageCustom)
	case "iacm":
		v.Spec = new(StageInfra)
	case "flag":
		v.Spec = new(StageFlag)
	case "template":
		v.Spec = new(StageTemplate)
	default:
		return fmt.Errorf("unknown type %s", v.Type)
	}

	return json.Unmarshal(obj.Spec, v.Spec)
}
