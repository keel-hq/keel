package kubernetes

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/version"

	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
)

func unsafeGetVersion(ver string) *types.Version {
	v, err := version.GetVersion(ver)
	if err != nil {
		panic(err)
	}
	return v
}

func TestProvider_checkVersionedDeployment(t *testing.T) {
	type fields struct {
		implementer     Implementer
		sender          notification.Sender
		approvalManager approvals.Manager
		events          chan *types.Event
		stop            chan struct{}
	}
	type args struct {
		newVersion *types.Version
		policy     types.PolicyType
		repo       *types.Repository
		deployment v1beta1.Deployment
	}
	tests := []struct {
		name                       string
		fields                     fields
		args                       args
		wantUpdatePlan             *UpdatePlan
		wantShouldUpdateDeployment bool
		wantErr                    bool
	}{
		{
			name: "standard version bump",
			args: args{
				newVersion: unsafeGetVersion("1.1.2"),
				policy:     types.PolicyTypeAll,
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.2"},
				deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:1.1.1",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
			},
			wantUpdatePlan: &UpdatePlan{
				Deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:1.1.2",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
				NewVersion:     "1.1.2",
				CurrentVersion: "1.1.1",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
		{
			name: "standard ignore version bump",
			args: args{
				newVersion: unsafeGetVersion("1.1.1"),
				policy:     types.PolicyTypeAll,
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.1"},
				deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:1.1.1",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
			},
			wantUpdatePlan: &UpdatePlan{
				Deployment:     v1beta1.Deployment{},
				NewVersion:     "",
				CurrentVersion: "",
			},
			wantShouldUpdateDeployment: false,
			wantErr:                    false,
		},
		{
			name: "multiple containers, version bump one",
			args: args{
				newVersion: unsafeGetVersion("1.1.2"),
				policy:     types.PolicyTypeAll,
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.2"},
				deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:1.1.1",
									},
									v1.Container{
										Image: "yo-world:1.1.1",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
			},
			wantUpdatePlan: &UpdatePlan{
				Deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:1.1.2",
									},
									v1.Container{
										Image: "yo-world:1.1.1",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
				NewVersion:     "1.1.2",
				CurrentVersion: "1.1.1",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
		{
			name: "force update untagged container",
			args: args{
				newVersion: unsafeGetVersion("1.1.2"),
				policy:     types.PolicyTypeForce,
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.2"},
				deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "force"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:latest",
									},
									v1.Container{
										Image: "yo-world:1.1.1",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
			},
			wantUpdatePlan: &UpdatePlan{
				Deployment: v1beta1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{forceUpdateImageAnnotation: "gcr.io/v2-namespace/hello-world:1.1.2"},
						Labels:      map[string]string{types.KeelPolicyLabel: "force"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:1.1.2",
									},
									v1.Container{
										Image: "yo-world:1.1.1",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
				NewVersion:     "1.1.2",
				CurrentVersion: "latest",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				implementer:     tt.fields.implementer,
				sender:          tt.fields.sender,
				approvalManager: tt.fields.approvalManager,
				events:          tt.fields.events,
				stop:            tt.fields.stop,
			}
			gotUpdatePlan, gotShouldUpdateDeployment, err := p.checkVersionedDeployment(tt.args.newVersion, tt.args.policy, tt.args.repo, tt.args.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.checkVersionedDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUpdatePlan, tt.wantUpdatePlan) {
				t.Errorf("Provider.checkVersionedDeployment() gotUpdatePlan = %v, want %v", gotUpdatePlan, tt.wantUpdatePlan)
			}
			if gotShouldUpdateDeployment != tt.wantShouldUpdateDeployment {
				t.Errorf("Provider.checkVersionedDeployment() gotShouldUpdateDeployment = %v, want %v", gotShouldUpdateDeployment, tt.wantShouldUpdateDeployment)
			}
		})
	}
}
