package kubernetes

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/types"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProvider_checkUnversionedDeployment(t *testing.T) {
	type fields struct {
		implementer     Implementer
		sender          notification.Sender
		approvalManager approvals.Manager
		events          chan *types.Event
		stop            chan struct{}
	}
	type args struct {
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
			name: "force update untagged to latest",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "latest"},
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
										Image: "gcr.io/v2-namespace/hello-world",
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
						Annotations: map[string]string{forceUpdateImageAnnotation: "gcr.io/v2-namespace/hello-world:latest"},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:latest",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
				NewVersion:     "latest",
				CurrentVersion: "latest",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
		{
			name: "different image name ",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "latest"},
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
										Image: "gcr.io/v2-namespace/goodbye-world:earliest",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
			},
			wantUpdatePlan: &UpdatePlan{
				Deployment: v1beta1.Deployment{},
			},
			wantShouldUpdateDeployment: false,
			wantErr:                    false,
		},
		{
			name: "dockerhub short image name ",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "karolisr/keel", Tag: "0.2.0"},
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
										Image: "karolisr/keel:latest",
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
						Annotations: map[string]string{forceUpdateImageAnnotation: "karolisr/keel:0.2.0"},
						Labels:      map[string]string{types.KeelPolicyLabel: "force"},
					},
					v1beta1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:0.2.0",
									},
								},
							},
						},
					},
					v1beta1.DeploymentStatus{},
				},
				NewVersion:     "0.2.0",
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
			gotUpdatePlan, gotShouldUpdateDeployment, err := p.checkUnversionedDeployment(tt.args.policy, tt.args.repo, tt.args.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.checkUnversionedDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUpdatePlan, tt.wantUpdatePlan) {
				t.Errorf("Provider.checkUnversionedDeployment() gotUpdatePlan = %v, want %v", gotUpdatePlan, tt.wantUpdatePlan)
			}
			if gotShouldUpdateDeployment != tt.wantShouldUpdateDeployment {
				t.Errorf("Provider.checkUnversionedDeployment() gotShouldUpdateDeployment = %v, want %v", gotShouldUpdateDeployment, tt.wantShouldUpdateDeployment)
			}
		})
	}
}
