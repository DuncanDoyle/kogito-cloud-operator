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

package framework

import (
	"fmt"
	"time"

	coreapps "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	rbac "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/kiegroup/kogito-cloud-operator/pkg/client/kubernetes"
	infra "github.com/kiegroup/kogito-cloud-operator/pkg/infrastructure"

	olmapiv1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1"
	olmapiv1alpha1 "github.com/operator-framework/operator-lifecycle-manager/pkg/api/apis/operators/v1alpha1"
)

const (
	kogitoOperatorTimeoutInMin = 5

	communityCatalog = "community-operators"
)

type dependentOperator struct {
	timeoutInMin int
	channel      string
}

var (
	kogitoOperatorCommunityDependencies = map[string]dependentOperator{
		"infinispan": {
			timeoutInMin: 20,
			channel:      "stable",
		},
		"strimzi-kafka-operator": {
			timeoutInMin: 20,
			channel:      "stable",
		},
		"keycloak-operator": {
			timeoutInMin: 20,
			channel:      "alpha",
		},
	}
)

// DeployKogitoOperatorFromYaml Deploy Kogito Operator from yaml files
func DeployKogitoOperatorFromYaml(namespace string) error {
	var deployURI = GetConfigOperatorDeployURI()
	GetLogger(namespace).Infof("Deploy Operator from yaml files in %s", deployURI)

	// TODO: error handling, go lint is screaming about this
	loadResource(namespace, deployURI+"service_account.yaml", &corev1.ServiceAccount{}, nil)
	loadResource(namespace, deployURI+"role.yaml", &rbac.Role{}, nil)
	loadResource(namespace, deployURI+"role_binding.yaml", &rbac.RoleBinding{}, nil)
	loadResource(namespace, deployURI+"operator.yaml", &coreapps.Deployment{}, func(object interface{}) {
		GetLogger(namespace).Debugf("Using operator image %s", getOperatorImageNameAndTag())
		object.(*coreapps.Deployment).Spec.Template.Spec.Containers[0].Image = getOperatorImageNameAndTag()
	})

	return nil
}

// IsKogitoOperatorRunning returns whether Kogito operator is running
func IsKogitoOperatorRunning(namespace string) (bool, error) {
	exists, err := infra.CheckKogitoOperatorExists(kubeClient, namespace)
	if err != nil {
		if exists {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

// WaitForKogitoOperatorRunning waits for Kogito operator running
func WaitForKogitoOperatorRunning(namespace string) error {
	return WaitFor(namespace, "Kogito operator running", time.Minute*time.Duration(kogitoOperatorTimeoutInMin), func() (bool, error) {
		return IsKogitoOperatorRunning(namespace)
	})
}

// WaitForKogitoOperatorRunningWithDependencies waits for Kogito operator running as well as other dependent operators
func WaitForKogitoOperatorRunningWithDependencies(namespace string) error {
	if err := WaitForKogitoOperatorRunning(namespace); err != nil {
		return err
	}
	return WaitForKogitoOperatorDependenciesRunning(namespace)
}

// InstallCommunityKogitoOperatorDependencies installs all dependent operators
func InstallCommunityKogitoOperatorDependencies(namespace string) error {
	for subscriptionName := range kogitoOperatorCommunityDependencies {
		operatorInfo := kogitoOperatorCommunityDependencies[subscriptionName]
		if err := InstallCommunityOperator(namespace, subscriptionName, operatorInfo.channel); err != nil {
			return err
		}
	}
	return nil
}

// WaitForKogitoOperatorDependenciesRunning waits for all dependent operators to be running
func WaitForKogitoOperatorDependenciesRunning(namespace string) error {
	for subscriptionName := range kogitoOperatorCommunityDependencies {
		operatorInfo := kogitoOperatorCommunityDependencies[subscriptionName]
		if err := WaitForOperatorRunning(namespace, subscriptionName, communityCatalog, operatorInfo.timeoutInMin); err != nil {
			return err
		}
	}
	return nil
}

// InstallCommunityOperator installs an operator from 'community-operators' catalog
func InstallCommunityOperator(namespace, subscriptionName, channel string) error {
	return InstallOperator(namespace, subscriptionName, communityCatalog, channel)
}

// InstallOperator installs an operator via subscrition
func InstallOperator(namespace, subscriptionName, operatorSource, channel string) error {
	GetLogger(namespace).Infof("Subscribing to %s operator from source %s on channel %s", subscriptionName, operatorSource, channel)
	if _, err := CreateOperatorGroupIfNotExists(namespace, namespace); err != nil {
		return err
	}

	if _, err := CreateNamespacedSubscriptionIfNotExist(namespace, subscriptionName, subscriptionName, operatorSource, channel); err != nil {
		return err
	}

	return nil
}

// WaitForOperatorRunning waits for an operator to be running
func WaitForOperatorRunning(namespace, operatorPackageName, operatorSource string, timeoutInMin int) error {
	return WaitFor(namespace, fmt.Sprintf("%s operator running", operatorPackageName), time.Minute*time.Duration(timeoutInMin), func() (bool, error) {
		return IsOperatorRunning(namespace, operatorPackageName, operatorSource)
	})
}

// IsOperatorRunning checks whether an operator is running
func IsOperatorRunning(namespace, operatorPackageName, operatorSource string) (bool, error) {
	exists, err := infra.CheckOperatorExistsUsingSubscription(kubeClient, namespace, operatorPackageName, operatorSource)
	if err != nil {
		if exists {
			return false, nil
		}
		return false, err
	}
	return exists, nil
}

// CreateOperatorGroupIfNotExists creates an operator group if no exist
func CreateOperatorGroupIfNotExists(namespace, operatorGroupName string) (*olmapiv1.OperatorGroup, error) {
	operatorGroup := &olmapiv1.OperatorGroup{
		ObjectMeta: metav1.ObjectMeta{
			Name:      operatorGroupName,
			Namespace: namespace,
		},
		Spec: olmapiv1.OperatorGroupSpec{
			TargetNamespaces: []string{namespace},
		},
	}
	if _, err := kubernetes.ResourceC(kubeClient).CreateIfNotExists(operatorGroup); err != nil {
		return nil, fmt.Errorf("Error creating OperatorGroup %s: %v", operatorGroupName, err)
	}
	return operatorGroup, nil
}

// CreateNamespacedSubscriptionIfNotExist create a namespaced subscription if not exists
func CreateNamespacedSubscriptionIfNotExist(namespace string, subscriptionName string, operatorName string, operatorSource string, channel string) (*olmapiv1alpha1.Subscription, error) {
	subscription := &olmapiv1alpha1.Subscription{
		ObjectMeta: metav1.ObjectMeta{
			Name:      subscriptionName,
			Namespace: namespace,
		},
		Spec: &olmapiv1alpha1.SubscriptionSpec{
			Package:                operatorName,
			CatalogSource:          operatorSource,
			CatalogSourceNamespace: "openshift-marketplace",
			Channel:                channel,
		},
	}
	if _, err := kubernetes.ResourceC(kubeClient).CreateIfNotExists(subscription); err != nil {
		return nil, fmt.Errorf("Error creating Subscription %s: %v", subscriptionName, err)
	}

	return subscription, nil
}

func getOperatorImageNameAndTag() string {
	return fmt.Sprintf("%s:%s", GetConfigOperatorImageName(), GetConfigOperatorImageTag())
}
