package kubernetes

import (
	"reflect"
	"testing"
)

func Test_addImageToPull(t *testing.T) {
	type args struct {
		annotations map[string]string
		image       string
	}
	tests := []struct {
		name string
		args args
		want map[string]string
	}{
		{
			name: "empty",
			args: args{annotations: make(map[string]string), image: "whatever"},
			want: map[string]string{forceUpdateImageAnnotation: "whatever"},
		},
		{
			name: "not empty",
			args: args{annotations: map[string]string{forceUpdateImageAnnotation: "foo"}, image: "bar"},
			want: map[string]string{forceUpdateImageAnnotation: "foo,bar"},
		},
		{
			name: "not empty with same image",
			args: args{annotations: map[string]string{forceUpdateImageAnnotation: "foo"}, image: "foo"},
			want: map[string]string{forceUpdateImageAnnotation: "foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := addImageToPull(tt.args.annotations, tt.args.image); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("addImageToPull() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_shouldPullImage(t *testing.T) {
	type args struct {
		annotations map[string]string
		image       string
	}
	tests := []struct {
		name string
		args args
		want bool
	}{
		{
			name: "should pull single image",
			args: args{annotations: map[string]string{forceUpdateImageAnnotation: "bar"}, image: "bar"},
			want: true,
		},
		{
			name: "should pull multiple image",
			args: args{annotations: map[string]string{forceUpdateImageAnnotation: "foo,bar,whatever"}, image: "bar"},
			want: true,
		},
		{
			name: "should not pull multiple image",
			args: args{annotations: map[string]string{forceUpdateImageAnnotation: "foo,bar,whatever"}, image: "alpha"},
			want: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := shouldPullImage(tt.args.annotations, tt.args.image); got != tt.want {
				t.Errorf("shouldPullImage() = %v, want %v", got, tt.want)
			}
		})
	}
}
