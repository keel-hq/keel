package helm

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/internal/policy"
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
		Values:   &hapi_chart.Config{Raw: chartValuesPolicyForce},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
	}

	helloWorldChartPolicyMajor := &hapi_chart.Chart{
		Values:   &hapi_chart.Config{Raw: chartValuesPolicyMajor},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
	}

	helloWorldChartPolicyMajorReleaseNotes := &hapi_chart.Chart{
		Values:   &hapi_chart.Config{Raw: chartValuesPolicyForceReleaseNotes},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
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
					Policy:          "force",
					MatchPreRelease: true,
					Trigger:         types.TriggerTypePoll,
					Images: []ImageDetails{
						{
							RepositoryPath: "image.repository",
							TagPath:        "image.tag",
						},
					},
					Plc: policy.NewForcePolicy(false),
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
					Policy:          "force",
					MatchPreRelease: true,
					Trigger:         types.TriggerTypePoll,
					Images: []ImageDetails{
						{
							RepositoryPath: "image.repository",
							TagPath:        "image.tag",
							ReleaseNotes:   "https://github.com/keel-hq/keel/releases",
						},
					},
					Plc: policy.NewForcePolicy(false),
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
			gotPlan, gotShouldUpdateRelease, err := checkRelease(tt.args.repo, tt.args.namespace, tt.args.name, tt.args.chart, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPlan, tt.wantPlan) {
				t.Errorf("checkRelease() gotPlan = %v, want %v", gotPlan, tt.wantPlan)
			}
			if gotShouldUpdateRelease != tt.wantShouldUpdateRelease {
				t.Errorf("checkRelease() gotShouldUpdateRelease = %v, want %v", gotShouldUpdateRelease, tt.wantShouldUpdateRelease)
			}
		})
	}
}

func Test_checkRelease(t *testing.T) {

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
		Values:   &hapi_chart.Config{Raw: chartValuesA},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
	}

	helloWorldNonSemverChart := &hapi_chart.Chart{
		Values:   &hapi_chart.Config{Raw: chartValuesB},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
	}
	helloWorldNonSemverNoForceChart := &hapi_chart.Chart{
		Values:   &hapi_chart.Config{Raw: chartValuesNonSemverNoForce},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
	}
	helloWorldNoTagChart := &hapi_chart.Chart{
		Values:   &hapi_chart.Config{Raw: chartValuesNoTag},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
	}

	helloWorldNoKeelCfg := &hapi_chart.Chart{
		Values:   &hapi_chart.Config{Raw: chartValuesNoKeelCfg},
		Metadata: &hapi_chart.Metadata{Name: "app-x"},
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
			name: "correct version bump",
			args: args{
				// newVersion: unsafeGetVersion("1.1.2"),
				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.2"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1",
				Chart:          helloWorldChart,
				Values:         map[string]string{"image.tag": "1.1.2"},
				NewVersion:     "1.1.2",
				CurrentVersion: "1.1.0",
				Config: &KeelChartConfig{
					Policy:          "all",
					MatchPreRelease: true,
					Trigger:         types.TriggerTypePoll,
					Images: []ImageDetails{
						{RepositoryPath: "image.repository", TagPath: "image.tag"},
					},
					Plc: policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "correct but same version",
			args: args{
				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1", Chart: helloWorldChart, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
		{
			name: "different image",
			args: args{

				repo:      &types.Repository{Name: "gcr.io/v2-namespace/bye-world", Tag: "1.1.5"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1", Chart: helloWorldChart, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
		{
			name: "non semver existing version",
			args: args{

				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldNonSemverChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1",
				Chart:          helloWorldNonSemverChart,
				Values:         map[string]string{"image.tag": "1.1.0"},
				NewVersion:     "1.1.0",
				CurrentVersion: "alpha",
				Config: &KeelChartConfig{
					Policy:          "force",
					MatchPreRelease: true,
					Trigger:         types.TriggerTypePoll,
					Images: []ImageDetails{
						{RepositoryPath: "image.repository", TagPath: "image.tag"},
					},
					Plc: policy.NewForcePolicy(false),
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "non semver no force, should not add to plan",
			args: args{

				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace: "default",
				name:      "release-1",
				chart:     helloWorldNonSemverNoForceChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1", Chart: helloWorldNonSemverNoForceChart, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
		{
			name: "semver no tag",
			args: args{

				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace: "default",
				name:      "release-1-no-tag",
				chart:     helloWorldNoTagChart,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan: &UpdatePlan{
				Namespace:      "default",
				Name:           "release-1-no-tag",
				Chart:          helloWorldNoTagChart,
				Values:         map[string]string{"image.repository": "gcr.io/v2-namespace/hello-world:1.1.0"},
				NewVersion:     "1.1.0",
				CurrentVersion: "1.0.0",
				Config: &KeelChartConfig{
					Policy:          "major",
					MatchPreRelease: true,
					Trigger:         types.TriggerTypePoll,
					Images: []ImageDetails{
						{RepositoryPath: "image.repository"},
					},
					Plc: policy.NewSemverPolicy(policy.SemverPolicyTypeMajor, true),
				},
			},
			wantShouldUpdateRelease: true,
			wantErr:                 false,
		},
		{
			name: "no keel config",
			args: args{

				repo:      &types.Repository{Name: "gcr.io/v2-namespace/hello-world", Tag: "1.1.0"},
				namespace: "default",
				name:      "release-1-no-tag",
				chart:     helloWorldNoKeelCfg,
				config:    &hapi_chart.Config{Raw: ""},
			},
			wantPlan:                &UpdatePlan{Namespace: "default", Name: "release-1-no-tag", Chart: helloWorldNoKeelCfg, Values: map[string]string{}},
			wantShouldUpdateRelease: false,
			wantErr:                 false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotPlan, gotShouldUpdateRelease, err := checkRelease(tt.args.repo, tt.args.namespace, tt.args.name, tt.args.chart, tt.args.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("checkRelease() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(gotPlan, tt.wantPlan) {
				t.Errorf("checkRelease() gotPlan = %v, want %v", gotPlan, tt.wantPlan)
			}
			if gotShouldUpdateRelease != tt.wantShouldUpdateRelease {
				t.Errorf("checkRelease() gotShouldUpdateRelease = %v, want %v", gotShouldUpdateRelease, tt.wantShouldUpdateRelease)
			}
		})
	}
}
