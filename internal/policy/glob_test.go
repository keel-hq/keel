package policy

import "testing"

func TestGlobPolicy_ShouldUpdate(t *testing.T) {
	type fields struct {
		policy  string
		pattern string
	}
	type args struct {
		current string
		new     string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    bool
		wantErr bool
	}{
		{
			name:    "test glob latest",
			fields:  fields{pattern: "latest"},
			args:    args{current: "latest", new: "latest"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "test glob without *",
			fields:  fields{pattern: "latest"},
			args:    args{current: "latest", new: "earliest"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "test glob with lat*",
			fields:  fields{pattern: "lat*"},
			args:    args{current: "latest", new: "latest"},
			want:    false,
			wantErr: false,
		},
		{
			name:    "test glob with latest.*",
			fields:  fields{pattern: "latest.*"},
			args:    args{current: "latest.20241321", new: "latest.20251321"},
			want:    true,
			wantErr: false,
		},
		{
			name:    "test glob with latest.* reverse",
			fields:  fields{pattern: "latest.*"},
			args:    args{current: "latest.20251321", new: "latest.20241321"},
			want:    false,
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &GlobPolicy{
				policy:  tt.fields.policy,
				pattern: tt.fields.pattern,
			}
			got, err := p.ShouldUpdate(tt.args.current, tt.args.new)
			if (err != nil) != tt.wantErr {
				t.Errorf("GlobPolicy.ShouldUpdate() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("GlobPolicy.ShouldUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}
