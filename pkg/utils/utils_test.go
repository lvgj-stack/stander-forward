package utils

import (
	"testing"
)

func TestGetOutBoundIP(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "test01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutBoundIPv4(); got != tt.want {
				t.Errorf("GetOutBoundIP() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestMd5Hash(t *testing.T) {
	type args struct {
		s string
	}
	tests := []struct {
		name string
		args args
		want string
	}{
		{
			name: "test 01",
			args: args{s: "123456"},
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := Md5Hash(tt.args.s); got != tt.want {
				t.Errorf("Md5Hash() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHandleTcpping(t *testing.T) {
	type args struct {
		destination string
	}
	tests := []struct {
		name    string
		args    args
		want    int
		wantErr bool
	}{
		{
			name: "test 01",
			args: args{destination: "[103.103.245.90]:30020"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := HandleTcpping(tt.args.destination)
			if (err != nil) != tt.wantErr {
				t.Errorf("HandleTcpping() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("HandleTcpping() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOutBoundIPv6(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "test 01",
			want: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutBoundIPv6(); got != tt.want {
				t.Errorf("GetOutBoundIPv6() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetOutBoundIPv4V2(t *testing.T) {
	tests := []struct {
		name string
		want string
	}{
		{
			name: "test 01",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := GetOutBoundIPv4V2(); got != tt.want {
				t.Errorf("GetOutBoundIPv4V2() = %v, want %v", got, tt.want)
			}
		})
	}
}
