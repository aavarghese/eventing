/*
Copyright 2020 The Knative Authors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package resources

import (
	"testing"

	"github.com/google/go-cmp/cmp"
	v1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"knative.dev/pkg/apis"
	"knative.dev/pkg/kmp"

	"knative.dev/eventing/pkg/apis/sources/v1beta1"
)

func TestMakeReceiveAdapter(t *testing.T) {
	src := &v1beta1.PingSource{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "source-name",
			Namespace: "source-namespace",
			UID:       "source-uid",
		},
		Spec: v1beta1.PingSourceSpec{
			Schedule: "*/2 * * * *",
			JsonData: "data",
		},
	}

	args := Args{
		ServiceAccountName: "test-sa",
		AdapterName:        "test-name",
		Image:              "test-image",
		MetricsConfig:      "metrics",
		LoggingConfig:      "logging",
		NoShutdownAfter:    40,
	}

	want := &v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "apps/v1",
			Kind:       "Deployments",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "source-namespace",
			Name:      fmt.Sprintf("pingsource-%s-%s", src.Name, src.UID),
			Labels: map[string]string{
				"test-key1": "test-value1",
				"test-key2": "test-value2",
			},
			OwnerReferences: []metav1.OwnerReference{{
				APIVersion:         "sources.knative.dev/v1beta1",
				Kind:               "PingSource",
				Name:               src.Name,
				UID:                src.UID,
				Controller:         &yes,
				BlockOwnerDeletion: &yes,
			}},
		},
		Spec: v1.DeploymentSpec{
			Replicas: &replicas,
			Selector: &metav1.LabelSelector{
				MatchLabels: mtlabels,
			},
			Template: corev1.PodTemplateSpec{
				ObjectMeta: metav1.ObjectMeta{
					Labels: mtlabels,
				},
				Spec: corev1.PodSpec{
					ServiceAccountName: args.ServiceAccountName,
					Containers: []corev1.Container{
						{
							Name:  "dispatcher",
							Image: args.Image,
							Env: []corev1.EnvVar{{
								Name:  system.NamespaceEnvKey,
								Value: system.Namespace(),
							}, {
								Name: "NAMESPACE",
								ValueFrom: &corev1.EnvVarSource{
									FieldRef: &corev1.ObjectFieldSelector{
										FieldPath: "metadata.namespace",
									},
								},
							}, {
								Name:  "K_METRICS_CONFIG",
								Value: "metrics",
							}, {
								Name:  "K_LOGGING_CONFIG",
								Value: "logging",
							}, {
								Name:  "K_LEADER_ELECTION_CONFIG",
								Value: "",
							}, {
								Name:  "K_NO_SHUTDOWN_AFTER",
								Value: "40",
							}},
							// Set low resource requests and limits.
							// This should be configurable.
							Resources: corev1.ResourceRequirements{
								Requests: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("125m"),
									corev1.ResourceMemory: resource.MustParse("64Mi"),
								},
								Limits: corev1.ResourceList{
									corev1.ResourceCPU:    resource.MustParse("1000m"),
									corev1.ResourceMemory: resource.MustParse("2048Mi"),
								},
							},
							Ports: []corev1.ContainerPort{{
								Name:          "metrics",
								ContainerPort: 9090,
							}},
						},
					},
				},
			},
		},
	}

	got := MakeReceiveAdapter(args)

	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("unexpected condition (-want, +got) = %v", diff)
	}
}
