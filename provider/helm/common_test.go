package helm

import (
	"reflect"
	"testing"

	"github.com/rusenask/keel/util/image"
	"k8s.io/helm/pkg/chartutil"
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

func Test_getImages(t *testing.T) {
	vals, _ := chartutil.ReadValues([]byte(chartValuesA))
	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.0")

	type args struct {
		vals chartutil.Values
	}
	tests := []struct {
		name    string
		args    args
		want    []*image.Reference
		wantErr bool
	}{
		{
			name: "hello-world image",
			args: args{
				vals: vals,
			},
			want:    []*image.Reference{img},
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
