package helm

import (
	"reflect"
	"testing"

	"github.com/rusenask/keel/types"
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

var chartValuesBSecret = `
--- 
image: 
  imagePullSecret: muchsecrecy
  repository: gcr.io/v2-namespace/hello-world
  tag: "1.1.0"
keel: 
  images: 
    - 
      imagePullSecret: image.imagePullSecret
      repository: image.repository
      tag: image.tag
  policy: all
  trigger: poll
name: "al Rashid"
where: 
  city: Basrah
  title: caliph`

var chartValuesBSecretNoPath = `
--- 
image:   
  repository: gcr.io/v2-namespace/hello-world
  tag: "1.1.0"
keel: 
  images: 
    - 
      imagePullSecret: image.imagePullSecret
      repository: image.repository
      tag: image.tag
  policy: all
  trigger: poll
name: "al Rashid"
where: 
  city: Basrah
  title: caliph`

func Test_getImages(t *testing.T) {
	vals, _ := chartutil.ReadValues([]byte(chartValuesA))
	img, _ := image.Parse("gcr.io/v2-namespace/hello-world:1.1.0")

	valsSecret, err := chartutil.ReadValues([]byte(chartValuesBSecret))
	if err != nil {
		t.Fatalf("failed to parse chartValuesBSecret")
	}

	valsSecretNoPath, err := chartutil.ReadValues([]byte(chartValuesBSecretNoPath))
	if err != nil {
		t.Fatalf("failed to parse chartValuesBSecretNoPath")
	}

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
				&types.TrackedImage{
					Image:   img,
					Trigger: types.TriggerTypePoll,
				},
			},
			wantErr: false,
		},
		{
			name: "hello-world image with secrets",
			args: args{
				vals: valsSecret,
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image:   img,
					Trigger: types.TriggerTypePoll,
					Secrets: []string{"muchsecrecy"},
				},
			},
			wantErr: false,
		},
		{
			name: "hello-world image with secret but no actual value",
			args: args{
				vals: valsSecretNoPath,
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image:   img,
					Trigger: types.TriggerTypePoll,
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
