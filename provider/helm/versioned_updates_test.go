package helm

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/version"

	hapi_chart "k8s.io/helm/pkg/proto/hapi/chart"
)

func unsafeGetVersion(ver string) *types.Version {
	v, err := version.GetVersion(ver)
	if err != nil {
		panic(err)
	}
	return v
}

func Test_checkVersionedRelease(t *testing.T) {
	chartValuesA := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.1.0

keel:  
  policy: all  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag

`
	// non semver existing
	chartValuesB := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: alpha

keel:  
  policy: force  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag

`
	chartValuesNonSemverNoForce := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: alpha

keel:  
  policy: major  
  trigger: poll  
  images:
    - repository: image.repository
      tag: image.tag
`

	chartValuesNoTag := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world:1.0.0  

keel:  
  policy: major  
  trigger: poll  
  images:
    - repository: image.repository      
`

	chartValuesNoKeelCfg := `
name: al Rashid
where:
  city: Basrah
  title: caliph
image:
  repository: gcr.io/v2-namespace/hello-world
  tag: 1.0.0
`

	helloWorldChart := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesA},
	}

	helloWorldNonSemverChart := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesB},
	}
	helloWorldNonSemverNoForceChart := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesNonSemverNoForce},
	}
	helloWorldNoTagChart := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesNoTag},
	}

	helloWorldNoKeelCfg := &hapi_chart.Chart{
		Values: &hapi_chart.Config{Raw: chartValuesNoKeelCfg},
	}

	type args struct {
		newVersion *types.Version
		repo       *types.Repository
		namespace  string
		name       string
		chart      *hapi_chart.Chart
		config     *hapi_chart.Config
	}
	tests := []struct {
		name                    string
		args                    args
		wantPlan                *UpdatePlan
		wantShouldUpdateRelease bool
		wantErr                 bool
	}{
		{
			name: "correct version bump",
			args: args{
				newVersion: unsafeGetVersion("1.1.2"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.2"},
				namespace:  "default",
				name:       "release-1",
				chart:      helloWorldChart,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1",
				Chart:          helloWorldChart,
				Values:         map[string]string{"image.tag": "1.1.2"},
				NewVersion:     "1.1.2",
				CurrentVersion: "1.1.0",
				Config: &KeelChartConfig{
					Policy:  types.PolicyTypeAll,
					Trigger: types.TriggerTypePoll,
					Images: []ImageDetails{
						ImageDetails{RepositoryPath: "image.repository", TagPath: "image.tag"},
					},
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "correct but same version",
			args: args{
				newVersion: unsafeGetVersion("1.1.0"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace:  "default",
				name:       "release-1",
				chart:      helloWorldChart,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1", Chart: helloWorldChart, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
		{
			name: "different image",
			args: args{
				newVersion: unsafeGetVersion("1.1.5"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/bye-world", Tag: "1.1.5"},
				namespace:  "default",
				name:       "release-1",
				chart:      helloWorldChart,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1", Chart: helloWorldChart, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
		{
			name: "non semver existing version",
			args: args{
				newVersion: unsafeGetVersion("1.1.0"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace:  "default",
				name:       "release-1",
				chart:      helloWorldNonSemverChart,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1",
				Chart:          helloWorldNonSemverChart,
				Values:         map[string]string{"image.tag": "1.1.0"},
				NewVersion:     "1.1.0",
				CurrentVersion: "alpha",
				Config: &KeelChartConfig{
					Policy:  types.PolicyTypeForce,
					Trigger: types.TriggerTypePoll,
					Images: []ImageDetails{
						ImageDetails{RepositoryPath: "image.repository", TagPath: "image.tag"},
					},
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "non semver no force, should not add to plan",
			args: args{
				newVersion: unsafeGetVersion("1.1.0"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace:  "default",
				name:       "release-1",
				chart:      helloWorldNonSemverNoForceChart,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1", Chart: helloWorldNonSemverNoForceChart, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
		{
			name: "semver no tag",
			args: args{
				newVersion: unsafeGetVersion("1.1.0"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace:  "default",
				name:       "release-1-no-tag",
				chart:      helloWorldNoTagChart,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1-no-tag",
				Chart:          helloWorldNoTagChart,
				Values:         map[string]string{"image.repository": "gcr.io/v2-namespace/hello-world:1.1.0"},
				NewVersion:     "1.1.0",
				CurrentVersion: "1.0.0",
				Config: &KeelChartConfig{
					Policy:  types.PolicyTypeMajor,
					Trigger: types.TriggerTypePoll,
					Images: []ImageDetails{
						ImageDetails{RepositoryPath: "image.repository"},
					},
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "no keel config",
			args: args{
				newVersion: unsafeGetVersion("1.1.0"),
				repo:       &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace:  "default",
				name:       "release-1-no-tag",
				chart:      helloWorldNoKeelCfg,
				config:     &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1-no-tag", Chart: helloWorldNoKeelCfg, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlan, gotShouldUpdateRelease, err := checkVersionedRelease(tt.args.newVersion, tt.args.repo, tt.args.namespace, tt.args.name, tt.args.chart, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkVersionedRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPlan, tt.wantPlan) {
				t.Errorf("checkVersionedRelease() gotPlan = %v, want %v", gotPlan, tt.wantPlan)
			}
			if gotShouldUpdateRelease != tt.wantShouldUpdateRelease {
				t.Errorf("checkVersionedRelease() gotShouldUpdateRelease = %v, want %v", gotShouldUpdateRelease, tt.wantShouldUpdateRelease)
			}
		})
	}
}
