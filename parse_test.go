package prox5

import (
	"strings"
	"testing"
)

func Test_filter(t *testing.T) {
	type args struct {
		in string
	}
	type test struct {
		name         string
		args         args
		wantFiltered string
		wantOk       bool
	}
	var tests = []test{
		{
			name: "simple",
			args: args{
				in: "127.0.0.1:1080",
			},
			wantFiltered: "127.0.0.1:1080",
			wantOk:       true,
		},
		{
			name: "withAuth",
			args: args{
				in: "127.0.0.1:1080:user:pass",
			},
			wantFiltered: "user:pass@127.0.0.1:1080",
			wantOk:       true,
		},
		{
			name: "withAuthAlt",
			args: args{
				in: "user:pass@127.0.0.1:1080",
			},
			wantFiltered: "user:pass@127.0.0.1:1080",
			wantOk:       true,
		},
		{
			name: "simpleDomain",
			args: args{
				in: "yeet.com:1080",
			},
			wantFiltered: "yeet.com:1080",
			wantOk:       true,
		},
		{
			name: "simpleDomainWithAlpha",
			args: args{
				in: "YeEt.com:1080",
			},
			wantFiltered: "yeet.com:1080",
			wantOk:       true,
		},
		{
			name: "domainWithAuth",
			args: args{
				in: "yeet.com:1080:user:pass",
			},
			wantFiltered: "user:pass@yeet.com:1080",
			wantOk:       true,
		},
		{
			name: "domainWithAuthWithAlpha",
			args: args{
				in: "YeEt.com:1080:uSer:pAss",
			},
			wantFiltered: "uSer:pAss@yeet.com:1080",
			wantOk:       true,
		},
		{
			name: "ipv6",
			args: args{
				in: "[fe80::2ef0:5dff:fe7f:c299]:1080",
			},
			wantFiltered: "[fe80::2ef0:5dff:fe7f:c299]:1080",
			wantOk:       true,
		},
		{
			name: "ipv6WithAuth",
			args: args{
				in: "[fe80::2ef0:5dff:fe7f:c299]:1080:user:pass",
			},
			wantFiltered: "user:pass@[fe80::2ef0:5dff:fe7f:c299]:1080",
			wantOk:       true,
		},
		{
			name: "invalid",
			args: args{
				in: "yeet",
			},
			wantFiltered: "",
			wantOk:       false,
		},
	}

	var prefixTests = []test{{
		name: "invalid prefix",
		args: args{
			in: "yeet://yeet.com:1080",
		},
		wantFiltered: "",
		wantOk:       false,
	}}

	for protoStr, protoPrefix := range protoStrs {
		for _, tt := range tests {
			if strings.Contains(tt.name, "invalid") || strings.Contains(tt.name, "prefix") {
				continue
			}
			tt.args.in = protoPrefix + tt.args.in
			tt.name = tt.name + " with " + protoStr + " prefix"
			prefixTests = append(prefixTests, test{
				name:         tt.name,
				args:         tt.args,
				wantFiltered: protoPrefix + tt.wantFiltered,
				wantOk:       tt.wantOk,
			})
		}
	}

	t.Run("tests without protocol prefixes", func(t *testing.T) {
		for _, tt := range tests {
			t.Run(tt.name, func(t *testing.T) {
				gotFiltered, gotOk := filter(tt.args.in)
				if gotFiltered != tt.wantFiltered {
					t.Errorf("filter() gotFiltered = %v, want %v", gotFiltered, tt.wantFiltered)
				}
				if gotOk != tt.wantOk {
					t.Errorf("filter() gotOk = %v, want %v", gotOk, tt.wantOk)
				}
			})
		}
	})

	t.Run("tests with protocol prefixes", func(t *testing.T) {
		for _, tt := range prefixTests {
			t.Run(tt.name, func(t *testing.T) {
				gotFiltered, gotOk := filter(tt.args.in)
				if gotFiltered != tt.wantFiltered {
					t.Errorf("filter() gotFiltered = %v, want %v", gotFiltered, tt.wantFiltered)
				}
				if gotOk != tt.wantOk {
					t.Errorf("filter() gotOk = %v, want %v", gotOk, tt.wantOk)
				}
			})
		}
	})
}
