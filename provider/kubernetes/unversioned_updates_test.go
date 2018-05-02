package kubernetes

import (
	"reflect"
	"testing"
	"time"

	"github.com/keel-hq/keel/approvals"
	"github.com/keel-hq/keel/extension/notification"
	"github.com/keel-hq/keel/internal/k8s"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/timeutil"

	apps_v1 "k8s.io/api/apps/v1"
	"k8s.io/api/core/v1"
	meta_v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestProvider_checkUnversionedDeployment(t *testing.T) {

	timeutil.Now = func() time.Time {
		return time.Date(0, 0, 0, 0, 0, 0, 0, time.UTC)
	}
	defer func() { timeutil.Now = time.Now }()

	type fields struct {
		implementer     Implementer
		sender          notification.Sender
		approvalManager approvals.Manager
		events          chan *types.Event
		stop            chan struct{}
	}
	type args struct {
		policy   types.PolicyType
		repo     *types.Repository
		resource *k8s.GenericResource
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
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:latest",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
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
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "all"},
					},
					apps_v1.DeploymentSpec{
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
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				// Resource: &k8s.GenericResource{},
				Resource: nil,
			},
			wantShouldUpdateDeployment: false,
			wantErr:                    false,
		},
		{
			name: "different tag name for poll image",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "master"},
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Annotations: map[string]string{
							types.KeelPollScheduleAnnotation: types.KeelPollDefaultSchedule,
						},
						Labels: map[string]string{
							types.KeelPolicyLabel: "all",
						},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "gcr.io/v2-namespace/hello-world:alpha",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: nil,
			},
			wantShouldUpdateDeployment: false,
			wantErr:                    false,
		},
		{
			name: "dockerhub short image name ",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "karolisr/keel", Tag: "0.2.0"},
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "force"},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:latest",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{},
						Labels:      map[string]string{types.KeelPolicyLabel: "force"},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:0.2.0",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
				NewVersion:     "0.2.0",
				CurrentVersion: "latest",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
		{
			name: "poll trigger, same tag",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "karolisr/keel", Tag: "master"},
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:        "dep-1",
						Namespace:   "xxxx",
						Annotations: map[string]string{types.KeelPollScheduleAnnotation: types.KeelPollDefaultSchedule},
						Labels:      map[string]string{types.KeelPolicyLabel: "force"},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:master",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Annotations: map[string]string{
							types.KeelPollScheduleAnnotation: types.KeelPollDefaultSchedule,
						},
						Labels: map[string]string{types.KeelPolicyLabel: "force"},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:master",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
				NewVersion:     "master",
				CurrentVersion: "master",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},

		{
			name: "pubsub trigger, force-match, same tag",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "karolisr/keel", Tag: "latest-staging"},
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels: map[string]string{
							types.KeelPolicyLabel:        "force",
							types.KeelForceTagMatchLabel: "true",
						},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:latest-staging",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels: map[string]string{
							types.KeelForceTagMatchLabel: "true",
							types.KeelPolicyLabel:        "force",
						},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:latest-staging",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
				NewVersion:     "latest-staging",
				CurrentVersion: "latest-staging",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},

		{
			name: "pubsub trigger, force-match, same tag on eu.gcr.io",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Host: "eu.gcr.io", Name: "karolisr/keel", Tag: "latest-staging"},
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels: map[string]string{
							types.KeelPolicyLabel:        "force",
							types.KeelForceTagMatchLabel: "true",
						},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "eu.gcr.io/karolisr/keel:latest-staging",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels: map[string]string{
							types.KeelForceTagMatchLabel: "true",
							types.KeelPolicyLabel:        "force",
						},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
									// "time": timeutil.Now().String(),
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "eu.gcr.io/karolisr/keel:latest-staging",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
				NewVersion:     "latest-staging",
				CurrentVersion: "latest-staging",
			},
			wantShouldUpdateDeployment: true,
			wantErr:                    false,
		},
		{
			name: "pubsub trigger, force-match, different tag",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Name: "karolisr/keel", Tag: "latest-staging"},
				resource: MustParseGR(&apps_v1.Deployment{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels: map[string]string{
							types.KeelPolicyLabel:        "force",
							types.KeelForceTagMatchLabel: "true",
						},
					},
					apps_v1.DeploymentSpec{
						Template: v1.PodTemplateSpec{
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "karolisr/keel:latest-acceptance",
									},
								},
							},
						},
					},
					apps_v1.DeploymentStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: nil,
			},
			wantShouldUpdateDeployment: false,
			wantErr:                    false,
		},
		{
			name: "pubsub trigger, force-match, same tag on eu.gcr.io, daemonset",
			args: args{
				policy: types.PolicyTypeForce,
				repo:   &types.Repository{Host: "eu.gcr.io", Name: "karolisr/keel", Tag: "latest-staging"},
				resource: MustParseGR(&apps_v1.DaemonSet{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Labels:    map[string]string{types.KeelPolicyLabel: "force"},
						Annotations: map[string]string{
							types.KeelForceTagMatchLabel: "true",
						},
					},
					apps_v1.DaemonSetSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "eu.gcr.io/karolisr/keel:latest-staging",
									},
								},
							},
						},
					},
					apps_v1.DaemonSetStatus{},
				}),
			},
			wantUpdatePlan: &UpdatePlan{
				Resource: MustParseGR(&apps_v1.DaemonSet{
					meta_v1.TypeMeta{},
					meta_v1.ObjectMeta{
						Name:      "dep-1",
						Namespace: "xxxx",
						Annotations: map[string]string{
							types.KeelForceTagMatchLabel: "true",
						},
						Labels: map[string]string{types.KeelPolicyLabel: "force"},
					},
					apps_v1.DaemonSetSpec{
						Template: v1.PodTemplateSpec{
							ObjectMeta: meta_v1.ObjectMeta{
								Annotations: map[string]string{
									"this": "that",
									// "time": timeutil.Now().String(),
								},
							},
							Spec: v1.PodSpec{
								Containers: []v1.Container{
									v1.Container{
										Image: "eu.gcr.io/karolisr/keel:latest-staging",
									},
								},
							},
						},
					},
					apps_v1.DaemonSetStatus{},
				}),
				NewVersion:     "latest-staging",
				CurrentVersion: "latest-staging",
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
			gotUpdatePlan, gotShouldUpdateDeployment, err := p.checkUnversionedDeployment(tt.args.policy, tt.args.repo, tt.args.resource)
			if (err != nil) != tt.wantErr {
				t.Errorf("Provider.checkUnversionedDeployment() error = %#v, wantErr %#v", err, tt.wantErr)
				return
			}

			if gotShouldUpdateDeployment {
				ann := gotUpdatePlan.Resource.GetSpecAnnotations()

				if ann[types.KeelUpdateTimeAnnotation] != "" {
					delete(ann, types.KeelUpdateTimeAnnotation)
					gotUpdatePlan.Resource.SetSpecAnnotations(ann)
				} else {
					t.Errorf("Provider.checkUnversionedDeployment() missing types.KeelUpdateTimeAnnotation annotation")
				}
			}

			if !reflect.DeepEqual(gotUpdatePlan, tt.wantUpdatePlan) {
				t.Errorf("Provider.checkUnversionedDeployment() gotUpdatePlan = %#v, want %#v", gotUpdatePlan, tt.wantUpdatePlan)
			}
			if gotShouldUpdateDeployment != tt.wantShouldUpdateDeployment {
				t.Errorf("Provider.checkUnversionedDeployment() gotShouldUpdateDeployment = %#v, want %#v", gotShouldUpdateDeployment, tt.wantShouldUpdateDeployment)
			}
		})
	}
}
