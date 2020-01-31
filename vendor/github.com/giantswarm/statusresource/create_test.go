package statusresource

import (
	"testing"
	"time"

	providerv1alpha1 "github.com/giantswarm/apiextensions/pkg/apis/provider/v1alpha1"
	"github.com/google/go-cmp/cmp"
)

func Test_removeTimesFromNodes(t *testing.T) {
	testCases := []struct {
		name          string
		nodes         []providerv1alpha1.StatusClusterNode
		expectedNodes []providerv1alpha1.StatusClusterNode
	}{
		{
			name: "case 0: Ensure that name, version and labels are kept when dropping last transitioning time",
			nodes: []providerv1alpha1.StatusClusterNode{
				{
					Labels: map[string]string{
						"foo": "bar",
						"bar": "baz",
						"baz": "foo",
					},

					LastTransitionTime: providerv1alpha1.DeepCopyTime{
						Time: time.Unix(123000, 0),
					},
					Name:    "worker-1234",
					Version: "1.2.3",
				},
				{
					Labels: map[string]string{
						"abc": "123",
						"def": "456",
						"ghi": "789",
					},

					LastTransitionTime: providerv1alpha1.DeepCopyTime{
						Time: time.Unix(234500, 0),
					},
					Name:    "worker-2345",
					Version: "2.3.4",
				},
			},
			expectedNodes: []providerv1alpha1.StatusClusterNode{
				{
					Labels: map[string]string{
						"foo": "bar",
						"bar": "baz",
						"baz": "foo",
					},

					LastTransitionTime: providerv1alpha1.DeepCopyTime{
						Time: time.Time{},
					},
					Name:    "worker-1234",
					Version: "1.2.3",
				},
				{
					Labels: map[string]string{
						"abc": "123",
						"def": "456",
						"ghi": "789",
					},

					LastTransitionTime: providerv1alpha1.DeepCopyTime{
						Time: time.Time{},
					},
					Name:    "worker-2345",
					Version: "2.3.4",
				},
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			nodes := removeTimesFromNodes(tc.nodes)
			if diff := cmp.Diff(nodes, tc.expectedNodes); diff != "" {
				t.Fatalf("after removeTimesFromNodes() nodes differ from expected: (-got +want)\n%s", diff)
			}
		})
	}
}
