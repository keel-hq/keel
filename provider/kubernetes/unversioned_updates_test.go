package kubernetes

import (
	"reflect"
	"testing"

	"github.com/rusenask/keel/types"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/pkg/api/v1"
	"k8s.io/client-go/pkg/apis/extensions/v1beta1"
)

func TestProvider_checkUnversionedDeployment(t *testing.T) {
	type fields struct {
		implementer Implementer
		events      chan *types.Event
		stop        chan struct{}
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
		wantUpdated                v1beta1.Deployment
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
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels:    map[string]string{types.KeelPolicyLabel: "all"},
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
			wantUpdated: v1beta1.Deployment{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{
					Name:      "dep-1",
					Namespace: "xxxx",
					Labels:    map[string]string{types.KeelPolicyLabel: "all"},
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
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels:    map[string]string{types.KeelPolicyLabel: "all"},
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
			wantUpdated: v1beta1.Deployment{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{
					Name:      "dep-1",
					Namespace: "xxxx",
					Labels:    map[string]string{types.KeelPolicyLabel: "all"},
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
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels:    map[string]string{types.KeelPolicyLabel: "force"},
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
			wantUpdated: v1beta1.Deployment{
				meta_v1.TypeMeta{},
				meta_v1.ObjectMeta{
					Name:      "dep-1",
					Namespace: "xxxx",
					Labels:    map[string]string{types.KeelPolicyLabel: "force"},
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
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &Provider{
				implementer: tt.fields.implementer,
				events:      tt.fields.events,
				stop:        tt.fields.stop,
			}
			gotUpdated, gotShouldUpdateDeployment, err := p.checkUnversionedDeployment(tt.args.policy, tt.args.repo, tt.args.deployment)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.checkUnversionedDeployment() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotUpdated, tt.wantUpdated) {
				t.Errorf("Provider.checkUnversionedDeployment() gotUpdated = %v, want %v", gotUpdated, tt.wantUpdated)
			}
			if gotShouldUpdateDeployment != tt.wantShouldUpdateDeployment {
				t.Errorf("Provider.checkUnversionedDeployment() gotShouldUpdateDeployment = %v, want %v", gotShouldUpdateDeployment, tt.wantShouldUpdateDeployment)
			}
		})
	}
}
