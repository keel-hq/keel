package helm

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/types"
	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
)

func Test_checkUnversionedRelease(t *testing.T) {
	chartValuesPolicyForce := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.1.0

keel:  
  policy: force  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag

`
	chartValuesPolicyForceReleaseNotes := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.1.0

keel:  
  policy: force  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag
      releaseNotes: https://github.com/keel-hq/keel/releases			

`

	chartValuesPolicyMajor := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.1.0

keel:  
  policy: major  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag

`

	helloWorldChart := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesPolicyForce},
	}

	helloWorldChartPolicyMajor := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesPolicyMajor},
	}

	helloWorldChartPolicyMajorReleaseNotes := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesPolicyForceReleaseNotes},
	}

	type args struct {
		repo      *types.Repository
		namespace string
		name      string
		chart     *hapi_chart.Chart
		config    *hapi_chart.Config
	}
	tests := []struct {
		name                    string
		args                    args
		wantPlan                *UpdatePlan
		wantShouldUpdateRelease bool
		wantErr                 bool
	}{
		{
			name: "correct force update",
			args: args{
				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "latest"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1",
				Chart:          helloWorldChart,
				Values:         map[string]string{"image.tag": "latest"},
				CurrentVersion: "1.1.0",
				NewVersion:     "latest",
				Config: &KeelChartConfig{
					Policy:  types.PolicyTypeForce,
					Trigger: types.TriggerTypePoll,
					Images: []ImageDetails{
						ImageDetails{
							RepositoryPath: "image.repository",
							TagPath:        "image.tag",
						},
					},
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "correct force update, with release notes",
			args: args{
				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.2.0"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldChartPolicyMajorReleaseNotes,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1",
				Chart:          helloWorldChartPolicyMajorReleaseNotes,
				Values:         map[string]string{"image.tag": "1.2.0"},
				CurrentVersion: "1.1.0",
				NewVersion:     "1.2.0",
				ReleaseNotes:   []string{"https://github.com/keel-hq/keel/releases"},
				Config: &KeelChartConfig{
					Policy:  types.PolicyTypeForce,
					Trigger: types.TriggerTypePoll,
					Images: []ImageDetails{
						ImageDetails{
							RepositoryPath: "image.repository",
							TagPath:        "image.tag",
							ReleaseNotes:   "https://github.com/keel-hq/keel/releases",
						},
					},
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "update without force",
			args: args{
				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "latest"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldChartPolicyMajor,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace: "default",
				Name:      "release-1",
				Chart:     helloWorldChartPolicyMajor,
				Values:    map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlan, gotShouldUpdateRelease, err := checkUnversionedRelease(tt.args.repo, tt.args.namespace, tt.args.name, tt.args.chart, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkUnversionedRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPlan, tt.wantPlan) {
				t.Errorf("checkUnversionedRelease() gotPlan = %v, want %v", gotPlan, tt.wantPlan)
			}
			if gotShouldUpdateRelease != tt.wantShouldUpdateRelease {
				t.Errorf("checkUnversionedRelease() gotShouldUpdateRelease = %v, want %v", gotShouldUpdateRelease, tt.wantShouldUpdateRelease)
			}
		})
	}
}
