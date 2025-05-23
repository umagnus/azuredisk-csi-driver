/*
Copyright The Kubernetes Authors.

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

// Code generated by client-gen. DO NOT EDIT.

package fake

import (
	v1 "k8s.io/api/rbac/v1"
	rbacv1 "k8s.io/client-go/applyconfigurations/rbac/v1"
	gentype "k8s.io/client-go/gentype"
	typedrbacv1 "k8s.io/client-go/kubernetes/typed/rbac/v1"
)

// fakeClusterRoles implements ClusterRoleInterface
type fakeClusterRoles struct {
	*gentype.FakeClientWithListAndApply[*v1.ClusterRole, *v1.ClusterRoleList, *rbacv1.ClusterRoleApplyConfiguration]
	Fake *FakeRbacV1
}

func newFakeClusterRoles(fake *FakeRbacV1) typedrbacv1.ClusterRoleInterface {
	return &fakeClusterRoles{
		gentype.NewFakeClientWithListAndApply[*v1.ClusterRole, *v1.ClusterRoleList, *rbacv1.ClusterRoleApplyConfiguration](
			fake.Fake,
			"",
			v1.SchemeGroupVersion.WithResource("clusterroles"),
			v1.SchemeGroupVersion.WithKind("ClusterRole"),
			func() *v1.ClusterRole { return &v1.ClusterRole{} },
			func() *v1.ClusterRoleList { return &v1.ClusterRoleList{} },
			func(dst, src *v1.ClusterRoleList) { dst.ListMeta = src.ListMeta },
			func(list *v1.ClusterRoleList) []*v1.ClusterRole { return gentype.ToPointerSlice(list.Items) },
			func(list *v1.ClusterRoleList, items []*v1.ClusterRole) { list.Items = gentype.FromPointerSlice(items) },
		),
		fake,
	}
}
