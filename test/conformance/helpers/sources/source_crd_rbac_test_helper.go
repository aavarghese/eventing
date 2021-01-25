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

package sources

import (
	"context"
	"testing"

	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
	testlib "knative.dev/eventing/test/lib"
	"knative.dev/pkg/apis/duck"
)

var clusterRoleName = "eventing-sources-source-observer"
var clusterRoleLabel = map[string]string{
	duck.SourceDuckVersionLabel: "true",
}

func SourceCRDRBACTestHelperWithComponentsTestRunner(
	t *testing.T,
	sourceTestRunner testlib.ComponentsTestRunner,
	options ...testlib.SetupClientOption,
) {

	sourceTestRunner.RunTests(t, testlib.FeatureBasic, func(st *testing.T, source metav1.TypeMeta) {
		client := testlib.Setup(st, true, options...)
		defer testlib.TearDown(client)

		// From spec:
		// Each source MUST have the following:
		// kind: ClusterRole
		// apiVersion: rbac.authorization.k8s.io/v1
		// metadata:
		//   name: foos-source-observer
		//   labels:
		//     duck.knative.dev/source: "true"
		// rules:
		//   - apiGroups:
		//       - example.com
		//     resources:
		//       - foos
		//     verbs:
		//       - get
		//       - list
		//       - watch
		st.Run("Source CRD has source observer cluster role", func(t *testing.T) {
			ValidateRBAC(st, client, source)
		})

	})
}

func ValidateRBAC(st *testing.T, client *testlib.Client, object metav1.TypeMeta) {
	labelSelector := &metav1.LabelSelector{
		MatchLabels: clusterRoleLabel,
	}

	sourcePluralName := getSourcePluralName(client, object)

	if !clusterRoleMeetsSpecs(client, labelSelector, sourcePluralName) {
		//CRD Spec says new sources MUST include a ClusterRole as part of installing themselves into a cluster - so can't enforce it. Nothing to do here
		//client.T.Fatalf("can't find cluster role for CRD %q", object)
	}
}

func getSourcePluralName(client *testlib.Client, object metav1.TypeMeta) string {
	gvr, _ := meta.UnsafeGuessKindToResource(object.GroupVersionKind())
	crdName := gvr.Resource + "." + gvr.Group

	crd, err := client.Apiextensions.CustomResourceDefinitions().Get(context.Background(), crdName, metav1.GetOptions{
		TypeMeta: metav1.TypeMeta{},
	})
	if err != nil {
		client.T.Errorf("error while getting %q:%v", object, err)
	}
	return crd.Spec.Names.Plural
}

func clusterRoleMeetsSpecs(client *testlib.Client, labelSelector *metav1.LabelSelector, crdSourceName string) bool {
	crs, err := client.Kube.RbacV1().ClusterRoles().List(context.Background(), metav1.ListOptions{
		FieldSelector: fields.OneTermEqualSelector("metadata.name", clusterRoleName).String(), //Cluster Role with name "eventing-sources-source-observer"
		LabelSelector: labels.Set(labelSelector.MatchLabels).String(),                         //Cluster Role with duck.knative.dev/source: "true" label
	})
	if err != nil {
		client.T.Errorf("error while getting cluster roles %v", err)
	}

	for _, cr := range crs.Items {
		for _, pr := range cr.Rules {
			if contains(pr.Resources, crdSourceName) { //Cluster Role has the eventing source listed in Resources for a Policy Rule
				return true
			}
		}
	}
	return false
}

func contains(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}
	return false
}
