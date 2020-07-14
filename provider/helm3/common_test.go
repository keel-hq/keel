package helm3

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/internal/policy"
	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	"helm.sh/helm/v3/pkg/chartutil"
)

var chartValuesA = `
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

func mustParse(name string) *image.Reference {
	img, err := image.Parse(name)
	if err != nil {
		panic(err)
	}
	return img
}

func Test_getImages(t *testing.T) {
	vals, _ := chartutil.ReadValues([]byte(chartValuesA))
	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.0")

	promVals, _ := chartutil.ReadValues([]byte(promChartValues))

	type args struct {
		vals chartutil.Values
	}
	tests := []struct {
		name    string
		args    args
		want    []*types.TrackedImage
		wantErr bool
	}{
		{
			name: "hello-world image",
			args: args{
				vals: vals,
			},
			want: []*types.TrackedImage{
				{
					Image:   img,
					Trigger: types.TriggerTypePoll,
					Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
				},
			},
			wantErr: false,
		},
		{
			name: "prom config from https://raw.githubusercontent.com/helm/charts/master/stable/prometheus-operator/values.yaml",
			args: args{
				vals: promVals,
			},
			want: []*types.TrackedImage{
				{
					Image:   mustParse("quay.io/prometheus/alertmanager:v0.16.2"),
					Trigger: types.TriggerTypePoll,
					Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
				},
				{
					Image:   mustParse("quay.io/coreos/prometheus-operator:v0.29.0"),
					Trigger: types.TriggerTypePoll,
					Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
				},
				{
					Image:   mustParse("quay.io/prometheus/prometheus:v2.7.2"),
					Trigger: types.TriggerTypePoll,
					Policy:  policy.NewSemverPolicy(policy.SemverPolicyTypeAll, true),
				},
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			got, err := getImages(tt.args.vals)
			if (err != nil) != tt.wantErr {
				t.Errorf("getImages() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("getImages() = %v, want %v", got, tt.want)
			}
		})
	}
}

var promChartValues = `# Default values for prometheus-operator.
# This is a YAML-formatted file.
# Declare variables to be passed into your templates.

# keel config
keel:
  policy: all
  trigger: poll
  images:
  - repository: alertmanager.alertmanagerSpec.image.repository
    tag: alertmanager.alertmanagerSpec.image.tag
  - repository: prometheusOperator.image.repository
    tag: prometheusOperator.image.tag
  - repository: prometheus.prometheusSpec.image.repository
    tag: prometheus.prometheusSpec.image.tag

alertmanager:
  enabled: true
  alertmanagerSpec:
    image:
      repository: quay.io/prometheus/alertmanager
      tag: v0.16.2

prometheusOperator:
  enabled: true
  image:
    repository: quay.io/coreos/prometheus-operator
    tag: v0.29.0
    pullPolicy: IfNotPresent

prometheus:
  enabled: true
  prometheusSpec:
    image:
      repository: quay.io/prometheus/prometheus
      tag: v2.7.2
`
