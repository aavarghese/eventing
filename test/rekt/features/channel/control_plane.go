/*
Copyright 2021 The Knative Authors

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

package channel

import (
	"context"
	"fmt"
	"strings"

	authv1 "k8s.io/api/authorization/v1"
	apiextv1 "k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"knative.dev/eventing/pkg/apis/messaging"
	"knative.dev/eventing/test/rekt/resources/account_role"
	"knative.dev/eventing/test/rekt/resources/channel_impl"
	"knative.dev/pkg/apis/duck"
	apiextensionsclient "knative.dev/pkg/client/injection/apiextensions/client"
	kubeclient "knative.dev/pkg/client/injection/kube/client"
	"knative.dev/reconciler-test/pkg/environment"
	"knative.dev/reconciler-test/pkg/feature"
)

func ControlPlaneConformance() *feature.FeatureSet {
	fs := &feature.FeatureSet{
		Name: "Knative Channel Specification - Control Plane",
		Features: []feature.Feature{
			*ControlPlaneChannel(),
		},
	}

	return fs
}

func todo(ctx context.Context, t feature.T) {
	t.Log("TODO, Implement this.")
}

func noop(ctx context.Context, t feature.T) {}

func ControlPlaneChannel() *feature.Feature {
	f := feature.NewFeatureNamed("Conformance")

	sacmName := feature.MakeRandomK8sName("channelable-manipulator")
	f.Setup("Create Service Account for Channelable Manipulator",
		account_role.Install(sacmName, account_role.AsChannelableManipulator))

	saarName := feature.MakeRandomK8sName("addressale-resolver")
	f.Setup("Create Service Account for Addressable Resolver",
		account_role.Install(saarName, account_role.AsAddressableResolver))

	f.Stable("Aggregated Channelable Manipulator ClusterRole").
		Must("Every CRD MUST create a corresponding ClusterRole, that will be aggregated into the channelable-manipulator ClusterRole.",
			serviceAccountIsChannelableManipulator(sacmName)).
		Must("This ClusterRole MUST include permissions to create, get, list, watch, patch, and update the CRD's custom objects and their status.",
			noop). // Tested by serviceAccountIsChannelableManipulator
		Must("Each channel MUST have the duck.knative.dev/channelable: \"true\" label on its channelable-manipulator ClusterRole.",
			noop) // Tested by serviceAccountIsChannelableManipulator

	f.Stable("Aggregated Addressable Resolver ClusterRole").
		Must("Every CRD MUST create a corresponding ClusterRole, that will be aggregated into the addressable-resolver ClusterRole.",
			serviceAccountIsAddressableResolver(saarName)).
		Must("This ClusterRole MUST include permissions to get, list, and watch the CRD's custom objects and their status.",
			noop). // Tested by serviceAccountIsAddressableResolver
		Must("Each channel MUST have the duck.knative.dev/addressable: \"true\" label on its addressable-resolver ClusterRole.",
			todo) // Tested by serviceAccountIsAddressableResolver

	f.Stable("CustomResourceDefinition per Channel").
		Must("Each channel is namespaced", crdOfChannelIsNamespaced).
		Must("label of messaging.knative.dev/subscribable: true",
			crdOfChannelIsLabeled(messaging.SubscribableDuckVersionAnnotation, "true")).
		Must("label of duck.knative.dev/addressable: true",
			crdOfChannelIsLabeled(duck.AddressableDuckVersionLabel, "true")).
		Must("The category `channel`", crdOfChannelHasCategory("channel"))

	f.Stable("Annotation Requirements").
		Should("each instance SHOULD have annotation: messaging.knative.dev/subscribable: v1", todo)

	f.Stable("Spec Requirements").
		Must("Each channel CRD MUST contain an array of subscribers: spec.subscribers", todo)
		// Special note for Channel tests: The array of subscribers MUST NOT be set directly on the generic Channel custom object, but rather appended to the backing channel by the subscription itself.

	f.Stable("Status Requirements").
		Must("Each channel CRD MUST have a status subresource which contains [address]", todo).
		Must("Each channel CRD MUST have a status subresource which contains [subscribers (as an array)]", todo).
		Should("SHOULD have in status observedGeneration", todo).
		Must("observedGeneration MUST be populated if present", todo).
		Should("SHOULD have in status conditions (as an array)", todo).
		Should("status.conditions SHOULD indicate status transitions and error reasons if present", todo)

	f.Stable("Channel Status").
		Must("When the channel instance is ready to receive events status.address.url MUST be populated", todo).
		Must("When the channel instance is ready to receive events status.address.url status.addressable MUST be set to True", todo)

	f.Stable("Channel Subscriber Status").
		Must("The ready field of the subscriber identified by its uid MUST be set to True when the subscription is ready to be processed", todo)

	return f
}

// I want this for later:
//
//type EventingClient struct {
//	Channels messagingclientsetv1.ChannelInterface
//}
//
//func Client(ctx context.Context) *EventingClient {
//	mc := eventingclient.Get(ctx).MessagingV1()
//	env := environment.FromContext(ctx)
//
//	return &EventingClient{
//		Channels: mc.Channels(env.Namespace()),
//	}
//}
//
//const (
//	ChannelNameKey = "channelName"
//)
//
//func setChannelName(name string) feature.StepFn {
//	return func(ctx context.Context, t feature.T) {
//		state.SetOrFail(ctx, t, ChannelNameKey, name)
//	}
//}
//
//func getChannel(ctx context.Context, t feature.T) *messagingv1.Channel {
//	c := Client(ctx)
//	name := state.GetStringOrFail(ctx, t, ChannelNameKey)
//
//	channel, err := c.Channels.Get(ctx, name, metav1.GetOptions{})
//	if err != nil {
//		t.Errorf("failed to get Channel, %v", err)
//	}
//	return channel
//}

func crdOfChannel(ctx context.Context, t feature.T) *apiextv1.CustomResourceDefinition {
	gvr := channel_impl.GVR()
	name := strings.Join([]string{gvr.Resource, gvr.Group}, ".")

	crd, err := apiextensionsclient.Get(ctx).ApiextensionsV1().CustomResourceDefinitions().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		t.Log(err)
		t.FailNow()
	}
	return crd
}

func crdOfChannelIsNamespaced(ctx context.Context, t feature.T) {
	crd := crdOfChannel(ctx, t)

	if crd.Spec.Scope != apiextv1.NamespaceScoped {
		t.Logf("%q CRD is not namespaced", crd.Name)
		t.Fail()
	}
}

func crdOfChannelIsLabeled(key, want string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		crd := crdOfChannel(ctx, t)

		if got, found := crd.Labels[key]; !found {
			t.Logf("%q CRD does not have label %q", crd.Name, key)
			t.Fail()
		} else if got != want {
			t.Logf("%q CRD label %q expected to be %s, got %s", crd.Name, key, want, got)
			t.Fail()
		}
	}
}

func crdOfChannelHasCategory(want string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		crd := crdOfChannel(ctx, t)

		for _, got := range crd.Spec.Names.Categories {
			if got == want {
				// Success!
				return
			}
		}
		t.Logf("%q CRD does not have Category %q", crd.Name, want)
	}
}

// TODO: move this to framework?
func ServiceAccountSubjectAccessReviewAllowedOrFail(ctx context.Context, t feature.T, gvr schema.GroupVersionResource, subresource string, saName string, verb string) {
	env := environment.FromContext(ctx)
	kube := kubeclient.Get(ctx)

	r, err := kube.AuthorizationV1().SubjectAccessReviews().Create(ctx, &authv1.SubjectAccessReview{
		Spec: authv1.SubjectAccessReviewSpec{
			User: fmt.Sprintf("system:serviceaccount:%s:%s", env.Namespace(), saName),
			ResourceAttributes: &authv1.ResourceAttributes{
				Verb:        verb,
				Group:       gvr.Group,
				Version:     gvr.Version,
				Resource:    gvr.Resource,
				Subresource: subresource,
			},
		},
	}, metav1.CreateOptions{})
	if err != nil {
		t.Fatalf("Error while checking if %q is not allowed on %s.%s/%s subresource:%q. err: %q", verb, gvr.Resource, gvr.Group, gvr.Version, subresource, err)
	}
	if !r.Status.Allowed {
		t.Fatalf("Operation %q is not allowed on %s.%s/%s subresource:%q", verb, gvr.Resource, gvr.Group, gvr.Version, subresource)
	}
}

func serviceAccountIsChannelableManipulator(name string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		gvr := channel_impl.GVR()
		for _, verb := range []string{"get", "list", "watch", "update", "patch"} {
			ServiceAccountSubjectAccessReviewAllowedOrFail(ctx, t, gvr, "", name, verb)
			ServiceAccountSubjectAccessReviewAllowedOrFail(ctx, t, gvr, "status", name, verb)
		}

	}
}

func serviceAccountIsAddressableResolver(name string) feature.StepFn {
	return func(ctx context.Context, t feature.T) {
		gvr := channel_impl.GVR()
		for _, verb := range []string{"get", "list", "watch"} {
			ServiceAccountSubjectAccessReviewAllowedOrFail(ctx, t, gvr, "", name, verb)
			ServiceAccountSubjectAccessReviewAllowedOrFail(ctx, t, gvr, "status", name, verb)
		}

	}
}
