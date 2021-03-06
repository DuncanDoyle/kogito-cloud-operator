// Copyright 2019 Red Hat, Inc. and/or its affiliates
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

package shared

import (
	"github.com/kiegroup/kogito-cloud-operator/pkg/framework"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	"strings"

	"github.com/kiegroup/kogito-cloud-operator/pkg/apis/app/v1alpha1"
	"github.com/kiegroup/kogito-cloud-operator/pkg/util"
)

// FromStringArrayToControllerEnvs converts a string array in the format of key=value pairs to the required type for the KogitoApp controller
func FromStringArrayToControllerEnvs(strings []string) []v1alpha1.Env {
	if strings == nil {
		return nil
	}
	var envs []v1alpha1.Env
	mapStr := util.FromStringsKeyPairToMap(strings)
	for k, v := range mapStr {
		envs = append(envs, v1alpha1.Env{Name: k, Value: v})
	}
	return envs
}

// FromStringArrayToEnvs converts a string array in the format of key=value pairs to the required type for the Kubernetes EnvVar type
func FromStringArrayToEnvs(strings []string) []v1.EnvVar {
	if strings == nil {
		return nil
	}
	return framework.MapToEnvVar(util.FromStringsKeyPairToMap(strings))
}

// FromStringArrayToControllerResourceMap ...
func FromStringArrayToControllerResourceMap(strings []string) []v1alpha1.ResourceMap {
	if strings == nil {
		return nil
	}
	var res []v1alpha1.ResourceMap
	mapstr := util.FromStringsKeyPairToMap(strings)
	for k, v := range mapstr {
		res = append(res, v1alpha1.ResourceMap{Resource: v1alpha1.ResourceKind(k), Value: v})
	}
	return res
}

// FromStringArrayToResources ...
func FromStringArrayToResources(strings []string) v1.ResourceList {
	if strings == nil {
		return nil
	}
	res := v1.ResourceList{}
	mapStr := util.FromStringsKeyPairToMap(strings)
	for k, v := range mapStr {
		res[v1.ResourceName(k)] = resource.MustParse(v)
	}
	return res
}

// ExtractResource reads a string array in the format memory=512M, cpu=1 and returns the value for a given kind
func ExtractResource(kind v1alpha1.ResourceKind, resources []string) string {
	for _, res := range resources {
		resKV := strings.Split(res, "=")
		if string(kind) == resKV[0] {
			return resKV[1]
		}
	}

	return ""
}
