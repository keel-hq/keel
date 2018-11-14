package provider

import (
	"reflect"
	"testing"

	"github.com/keel-hq/keel/types"
	"github.com/keel-hq/keel/util/image"
	testingUtils "github.com/keel-hq/keel/util/testing"
)

func mustParse(i string) *image.Reference {
	ref, err := image.Parse(i)
	if err != nil {
		panic(err)
	}
	return ref
}

func Test_appendImage(t *testing.T) {
	type args struct {
		images []*types.TrackedImage
		new    *types.TrackedImage
	}
	tests := []struct {
		name string
		args args
		want []*types.TrackedImage
	}{
		{
			name: "new image",
			args: args{
				images: []*types.TrackedImage{},
				new:    testingUtils.GetTrackedImage("karolisr/webhook-demo:latest"),
			},
			want: []*types.TrackedImage{
				testingUtils.GetTrackedImage("karolisr/webhook-demo:latest"),
			},
		},
		{
			name: "new semver",
			args: args{
				images: []*types.TrackedImage{},
				new:    testingUtils.GetTrackedImage("karolisr/webhook-demo:1.2.3"),
			},
			want: []*types.TrackedImage{
				testingUtils.GetTrackedImage("karolisr/webhook-demo:1.2.3"),
			},
		},
		{
			name: "new semver with prerelease",
			args: args{
				images: []*types.TrackedImage{
					testingUtils.GetTrackedImage("karolisr/webhook-demo:1.2.3"),
				},
				new: testingUtils.GetTrackedImage("karolisr/webhook-demo:1.5.0-dev"),
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image: mustParse("karolisr/webhook-demo:1.5.0-dev"),
					SemverPreReleaseTags: map[string]string{
						"dev": "1.5.0-dev",
					},
					Meta: make(map[string]string),
					Tags: []string{"1.2.3", "1.5.0-dev"},
				},
			},
		},
		{
			name: "new semver with both prereleases",
			args: args{
				images: []*types.TrackedImage{
					&types.TrackedImage{
						Image: mustParse("karolisr/webhook-demo:1.2.3-prod"),
						SemverPreReleaseTags: map[string]string{
							"prod": "1.2.3-prod",
						},
						Meta: make(map[string]string),
						Tags: []string{"1.2.3-prod"},
					},
				},
				new: testingUtils.GetTrackedImage("karolisr/webhook-demo:1.5.0-dev"),
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image: mustParse("karolisr/webhook-demo:1.5.0-dev"),
					SemverPreReleaseTags: map[string]string{
						"dev":  "1.5.0-dev",
						"prod": "1.2.3-prod",
					},
					Meta: make(map[string]string),
					Tags: []string{"1.2.3-prod", "1.5.0-dev"},
				},
			},
		},
		{
			name: "semver prerelease",
			args: args{
				images: []*types.TrackedImage{},
				new:    testingUtils.GetTrackedImage("karolisr/webhook-demo:1.5.0-dev"),
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image: mustParse("karolisr/webhook-demo:1.5.0-dev"),
					SemverPreReleaseTags: map[string]string{
						"dev": "1.5.0-dev",
					},
					Meta: make(map[string]string),
					Tags: []string{"1.5.0-dev"},
				},
			},
		},
		{
			name: "new semver with previous non-semver tag",
			args: args{
				images: []*types.TrackedImage{
					&types.TrackedImage{
						Image:                mustParse("karolisr/webhook-demo:build-xx"),
						SemverPreReleaseTags: make(map[string]string),
						Meta:                 make(map[string]string),
					},
				},
				new: testingUtils.GetTrackedImage("karolisr/webhook-demo:1.5.0-dev"),
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image:                mustParse("karolisr/webhook-demo:build-xx"),
					SemverPreReleaseTags: make(map[string]string),
					Meta:                 make(map[string]string),
				},
				&types.TrackedImage{
					Image: mustParse("karolisr/webhook-demo:1.5.0-dev"),
					SemverPreReleaseTags: map[string]string{
						"dev": "1.5.0-dev",
					},
					Meta: make(map[string]string),
					Tags: []string{"1.5.0-dev"},
				},
			},
		},
		{
			name: "mixed versions",
			args: args{
				images: []*types.TrackedImage{
					&types.TrackedImage{
						Image:                mustParse("karolisr/webhook-demo:latest"),
						SemverPreReleaseTags: make(map[string]string),
						Meta:                 make(map[string]string),
					},
					&types.TrackedImage{
						Image:                mustParse("karolisr/webhook-demo:build-foo"),
						SemverPreReleaseTags: make(map[string]string),
						Meta:                 make(map[string]string),
					},
					&types.TrackedImage{
						Image: mustParse("karolisr/webhook-demo:1.5.0-prod"),
						SemverPreReleaseTags: map[string]string{
							"prod": "1.5.0-prod",
						},
						Meta: make(map[string]string),
						Tags: []string{"1.5.0-prod"},
					},
				},
				new: testingUtils.GetTrackedImage("karolisr/webhook-demo:1.7.0-dev"),
			},
			want: []*types.TrackedImage{
				&types.TrackedImage{
					Image:                mustParse("karolisr/webhook-demo:latest"),
					SemverPreReleaseTags: make(map[string]string),
					Meta:                 make(map[string]string),
				},
				&types.TrackedImage{
					Image:                mustParse("karolisr/webhook-demo:build-foo"),
					SemverPreReleaseTags: make(map[string]string),
					Meta:                 make(map[string]string),
				},
				&types.TrackedImage{
					Image: mustParse("karolisr/webhook-demo:1.7.0-dev"),
					SemverPreReleaseTags: map[string]string{
						"prod": "1.5.0-prod",
						"dev":  "1.7.0-dev",
					},
					Meta: make(map[string]string),
					Tags: []string{"1.5.0-prod", "1.7.0-dev"},
				},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := appendImage(tt.args.images, tt.args.new); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("appendImage() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_lookupSemverImageIdx(t *testing.T) {
	type args struct {
		images []*types.TrackedImage
		new    *types.TrackedImage
	}
	tests := []struct {
		name  string
		args  args
		want  int
		want1 bool
	}{
		{
			name: "different image",
			args: args{
				images: []*types.TrackedImage{
					testingUtils.GetTrackedImage("karolisr/webhook-demo:1.7.0-dev"),
				},
				new: testingUtils.GetTrackedImage("karolisr/foo:latest"),
			},
			want:  0,
			want1: false,
		},
		{
			name: "empty",
			args: args{
				images: []*types.TrackedImage{},
				new:    testingUtils.GetTrackedImage("karolisr/foo:latest"),
			},
			want:  0,
			want1: false,
		},
		{
			name: "semver second",
			args: args{
				images: []*types.TrackedImage{
					testingUtils.GetTrackedImage("karolisr/webhook-demo:dev"),
					testingUtils.GetTrackedImage("karolisr/webhook-demo:1.7.0-dev"),
					testingUtils.GetTrackedImage("karolisr/webhook-demo:test"),
				},
				new: testingUtils.GetTrackedImage("karolisr/webhook-demo:1.5.0"),
			},
			want:  1,
			want1: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := lookupSemverImageIdx(tt.args.images, tt.args.new)
			if got != tt.want {
				t.Errorf("lookupSemverImageIdx() got = %v, want %v", got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("lookupSemverImageIdx() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}
