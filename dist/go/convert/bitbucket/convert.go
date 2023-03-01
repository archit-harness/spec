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

package bitbucket

import (
	"fmt"
	"sort"
	"strings"
	"time"

	harness "github.com/drone/spec/dist/go"
	bitbucket "github.com/drone/spec/dist/go/convert/bitbucket/yaml"
	"github.com/gotidy/ptr"
)

func convertDefault(config *bitbucket.Config) *harness.Default {

	// if the global pipeline configuration sections
	// are empty or nil, return nil
	if config.Clone == nil &&
		config.Image == nil &&
		config.Options == nil {
		return nil
	}

	if config.Image == nil {
		// Username
		// Password
	}
	if config.Options == nil {
		// Docker (bool)
		// MaxTime (int)
		// Size (1x, 2x, 4x, 8x)
		// Credentials ???
	}

	var def *harness.Default

	// if the user has configured global clone defaults,
	// convert this to pipeline-level clone settings.
	if config.Clone != nil {
		// create the default if not already created.
		if def == nil {
			def = new(harness.Default)
		}
		def.Clone = convertCloneGlobal(config.Clone)

		// if the clone is disabled we need to make
		// sure it isn't explicitly enabled for any steps.
		if def.Clone.Disabled {
			for _, step := range extractAllSteps(config.Pipelines.Default) {
				if step.Clone != nil && ptr.ToBool(step.Clone.Enabled) {
					def.Clone.Disabled = false
					break
				}
			}
		}
	}

	return def
}

func convertPipeline() {
}

func convertStage(s *state) *harness.Stage {

	// create the harness stage spec
	spec := &harness.StageCI{
		Clone: convertClone(s.stage),
		// TODO Repository
		// TODO Delegate
		// TODO Platform
		// TODO Runtime
		// TODO Envs
	}

	// find the step with the largest size and use that
	// size. else fallback to the global size.
	if size := extractSize(s.config.Options, s.stage); size != bitbucket.SizeNone {
		spec.Runtime = &harness.Runtime{
			Type: "cloud",
			Spec: &harness.RuntimeCloud{
				Size: convertSize(size),
			},
		}
	}

	// find the unique cache paths used by this
	// stage and setup harness caching
	if paths := extractCache(s.stage); len(paths) != 0 {
		spec.Cache = convertCache(s.config.Definitions, paths)
	}

	// find the unique selectors and append
	// to the stage.
	if runson := extractRunsOn(s.stage); len(runson) != 0 {
		spec.Delegate = new(harness.Delegate)
		spec.Delegate.Selectors = runson
	}

	// create the harness stage.
	stage := &harness.Stage{
		Name: "build",
		Type: "ci",
		Spec: spec,
		// TODO When
		// TODO On
	}

	// default docker service (container-based only)
	if s.config.Options != nil && s.config.Options.Docker {
		spec.Steps = append(spec.Steps, &harness.Step{
			Name: s.generateName("dind", "service"),
			Type: "background",
			Spec: &harness.StepBackground{
				Image:      "docker:dind",
				Ports:      []string{"2375", "2376"},
				Privileged: true,
			},
		})
	}

	// default services
	// TODO

	for _, steps := range s.stage.Steps {
		if steps.Parallel != nil {
			// TODO parallel steps
			// TODO fast fail
			s.steps = steps // push the parallel step to the state
			step := convertParallel(s)
			spec.Steps = append(spec.Steps, step)
		}
		if steps.Step != nil {
			s.step = steps.Step // push the step to the state
			step := convertStep(s)
			spec.Steps = append(spec.Steps, step)
		}
	}

	// if the stage has a single step, and that step is a
	// group step, we can eliminate the un-necessary group
	// and add the steps directly to the stage.
	if len(spec.Steps) == 1 {
		if group, ok := spec.Steps[0].Spec.(*harness.StepGroup); ok {
			spec.Steps = group.Steps
		}
	}

	return stage
}

// helper function converts a bitbucket parallel step
// group to a Harness parallel step group.
func convertParallel(s *state) *harness.Step {

	// create the step group spec
	spec := new(harness.StepParallel)

	for _, src := range s.steps.Parallel.Steps {
		if src.Step != nil {
			s.step = src.Step
			step := convertStep(s)
			spec.Steps = append(spec.Steps, step)
		}
	}

	// else create the step group wrapper.
	return &harness.Step{
		Type: "parallel",
		Spec: spec,
		Name: s.generateName("parallel", "parallel"), // TODO can we avoid a name here?
	}
}

// helper function converts a bitbucket step
// to a harness run step or plugin step.
func convertStep(s *state) *harness.Step {
	// create the step group spec
	spec := new(harness.StepGroup)

	// loop through each script item
	for _, script := range s.step.Script {
		s.script = script

		// if a pipe step
		if script.Pipe != nil {
			step := convertPipeStep(s)
			spec.Steps = append(spec.Steps, step)
		}

		// else if a script step
		if script.Pipe == nil {
			step := convertScriptStep(s)
			spec.Steps = append(spec.Steps, step)
		}
	}

	// and loop through each after script item
	for _, script := range s.step.ScriptAfter {
		s.script = script

		// if a pipe step
		if script.Pipe != nil {
			step := convertPipeStep(s)
			spec.Steps = append(spec.Steps, step)
		}

		// else if a script step
		if script.Pipe == nil {
			step := convertScriptStep(s)
			spec.Steps = append(spec.Steps, step)
		}
	}

	// if there is only a single step, no need to
	// create a step group.
	if len(spec.Steps) == 1 {
		return spec.Steps[0]
	}

	// else create the step group wrapper.
	return &harness.Step{
		Type: "group",
		Spec: spec,
		Name: s.generateName(s.step.Name, "group"),
	}
}

// helper function converts a script step to a
// harness run step.
func convertScriptStep(s *state) *harness.Step {

	// create the run spec
	spec := &harness.StepExec{
		Run: s.script.Text,

		// TODO configure an optional connector
		// TODO configure pull policy
		// TODO configure envs
		// TODO configure volumes
		// TODO configure resources
	}

	// use the global image, if set
	if image := s.config.Image; image != nil {
		spec.Image = strings.TrimPrefix(image.Name, "docker://")
		if image.RunAsUser != 0 {
			spec.User = fmt.Sprint(image.RunAsUser)
		}
	}

	// use the step image, if set (overrides previous)
	if image := s.step.Image; image != nil {
		spec.Image = strings.TrimPrefix(image.Name, "docker://")
		if image.RunAsUser != 0 {
			spec.User = fmt.Sprint(image.RunAsUser)
		}
	}

	// create the run step wrapper
	step := &harness.Step{
		Type: "run",
		Spec: spec,
		Name: s.generateName(s.step.Name, "run"),
	}

	// use the global max-time, if set
	if s.config.Options != nil {
		if v := int64(s.config.Options.MaxTime); v != 0 {
			step.Timeout = minuteToDurationString(v)
		}
	}

	// set the timeout
	if v := int64(s.step.MaxTime); v != 0 {
		step.Timeout = minuteToDurationString(v)
	}

	return step
}

// helper function converts a pipe step to a
// harness plugin step.
func convertPipeStep(s *state) *harness.Step {
	pipe := s.script.Pipe

	// create the plugin spec
	spec := &harness.StepPlugin{
		Image: strings.TrimPrefix(pipe.Image, "docker://"),

		// TODO configure an optional connector
		// TODO configure envs
		// TODO configure volumes
	}

	// append the plugin spec variables
	spec.With = map[string]interface{}{}
	for key, val := range pipe.Variables {
		spec.With[key] = val
	}

	// create the plugin step wrapper
	step := &harness.Step{
		Type: "plugin",
		Spec: spec,
		Name: s.generateName(s.step.Name, "plugin"),
	}

	// set the timeout
	if v := int64(s.step.MaxTime); v != 0 {
		step.Timeout = minuteToDurationString(v)
	}

	return step
}

// helper function converts an integer of minutes
// to a time duration string.
func minuteToDurationString(v int64) string {
	dur := time.Duration(v) * time.Minute
	return fmt.Sprint(dur)
}

// helper function that returns a deep copy of all
// stage steps, including parallel steps.
func extractSteps(stage *bitbucket.Stage) []*bitbucket.Step {
	var steps []*bitbucket.Step
	// iterate through steps the he stage
	for _, step := range stage.Steps {
		if step.Step != nil {
			steps = append(steps, step.Step)
		}
		// iterate through parallel steps
		if step.Parallel != nil {
			for _, step2 := range step.Parallel.Steps {
				if step2.Step != nil {
					steps = append(steps, step2.Step)
				}
			}
		}
	}
	return steps
}

// helper function that returns a deep copy of all
// stage steps, including parallel steps.
func extractAllSteps(stages []*bitbucket.Steps) []*bitbucket.Step {
	var steps []*bitbucket.Step
	for _, stage := range stages {
		if stage.Stage != nil {
			steps = append(steps, extractSteps(stage.Stage)...)
		}
	}
	return steps
}

func extractSize(opts *bitbucket.Options, stage *bitbucket.Stage) bitbucket.Size {
	var size bitbucket.Size

	// start with the global size, if set
	if opts != nil {
		size = opts.Size
	}

	// loop through the steps and if a step
	// defines a size greater than the global
	// size, us the step size instead.
	for _, step := range extractSteps(stage) {
		if step.Size > size {
			size = step.Size
		}
	}
	return size
}

func extractRunsOn(stage *bitbucket.Stage) []string {
	set := map[string]struct{}{}

	// loop through the steps and if a step
	// defines a size greater than the global
	// size, us the step size instead.
	for _, step := range extractSteps(stage) {
		for _, s := range step.RunsOn {
			set[s] = struct{}{}
		}
	}

	// convert the map to a slice.
	var unique []string
	for k := range set {
		unique = append(unique, k)
	}

	// sort for deterministic unit testing
	sort.Strings(unique)

	return unique
}

func extractCache(stage *bitbucket.Stage) []string {
	set := map[string]struct{}{}

	// loop through the steps and if a step
	// defines cache directories
	for _, step := range extractSteps(stage) {
		for _, s := range step.Caches {
			set[s] = struct{}{}
		}
	}

	// convert the map to a slice.
	var unique []string
	for k := range set {
		unique = append(unique, k)
	}

	// sort for deterministic unit testing
	sort.Strings(unique)

	return unique
}

func convertClone(stage *bitbucket.Stage) *harness.Clone {
	var clones []*bitbucket.Clone

	// loop through the steps and if a step
	// defines cache directories
	for _, step := range extractSteps(stage) {
		if step.Clone != nil {
			clones = append(clones, step.Clone)
		}
	}

	// if there are not clone configurations at
	// the step-level we can return a nil clone.
	if len(clones) == 0 {
		return nil
	}

	clone := new(harness.Clone)
	for _, v := range clones {
		if v.Depth != nil {
			if v.Depth.Value > int(clone.Depth) {
				clone.Depth = int64(v.Depth.Value)
			}
		}
		if v.SkipVerify {
			clone.Insecure = true
		}
		if v.Enabled != nil && !ptr.ToBool(v.Enabled) {
			// TODO
		}
	}

	return clone
}

func convertSize(size bitbucket.Size) string {
	switch size {
	case bitbucket.Size2x: // 8GB
		return "large"
	case bitbucket.Size4x: // 16GB
		return "xlarge"
	case bitbucket.Size8x: // 32GB
		return "xxlarge"
	case bitbucket.Size1x: // 4GB
		return "standard"
	default:
		return ""
	}
}

func convertCache(defs *bitbucket.Definitions, caches []string) *harness.Cache {
	if defs == nil || len(defs.Caches) == 0 || len(caches) == 0 {
		return nil
	}

	cache := new(harness.Cache)
	cache.Enabled = true

	var files []string
	var paths []string

	for _, name := range caches {
		src, ok := defs.Caches[name]
		if !ok {
			continue
		}
		paths = append(paths, src.Path)
		if src.Key != nil {
			files = append(files, src.Key.Files...)
		}
	}

	for _, name := range caches {
		switch name {
		case "composer":
			paths = append(paths, "composer")
			paths = append(paths, "~/.composer/cache")
		case "dotnetcore":
			paths = append(paths, "dotnetcore")
			paths = append(paths, "~/.nuget/packages")
		case "gradle":
			paths = append(paths, "gradle")
			paths = append(paths, "~/.gradle/caches")
		case "ivy2":
			paths = append(paths, "ivy2")
			paths = append(paths, "~/.ivy2/cache")
		case "maven":
			paths = append(paths, "maven")
			paths = append(paths, "~/.m2/repository")
		case "node":
			paths = append(paths, "node")
			paths = append(paths, "node_modules")
		case "pip":
			paths = append(paths, "pip")
			paths = append(paths, "~/.cache/pip")
		case "sbt":
			paths = append(paths, "sbt")
			paths = append(paths, "ivy2")
			paths = append(paths, "~/.ivy2/cache")
		}
	}

	cache.Paths = paths
	return cache
}

func convertCloneGlobal(clone *bitbucket.Clone) *harness.Clone {
	if clone == nil {
		return nil
	}

	to := new(harness.Clone)
	to.Insecure = clone.SkipVerify

	if clone.Depth != nil {
		to.Depth = int64(clone.Depth.Value)
	}

	// disable cloning globally if the user has
	// explicityly disabled this functionality
	if clone.Enabled != nil && ptr.ToBool(clone.Enabled) == false {
		to.Disabled = true
	}

	return to
}
