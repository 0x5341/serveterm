package main

import (
	"net/http/httptest"
	"reflect"
	"testing"
)

func TestCommandSpecFromRequest_UsesQueryCommandAndArgs(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws?command=/bin/sh&arg=-lc&arg=echo+ok", nil)

	spec, err := commandSpecFromRequest(req, commandSpec{Command: "/bin/bash"})
	if err != nil {
		t.Fatalf("commandSpecFromRequest() error = %v", err)
	}

	if spec.Command != "/bin/sh" {
		t.Fatalf("command = %q, want %q", spec.Command, "/bin/sh")
	}
	if !reflect.DeepEqual(spec.Args, []string{"-lc", "echo ok"}) {
		t.Fatalf("args = %#v, want %#v", spec.Args, []string{"-lc", "echo ok"})
	}
}

func TestCommandSpecFromRequest_ParsesCommandString(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws?command=/bin/sh+-lc+echo", nil)

	spec, err := commandSpecFromRequest(req, commandSpec{Command: "/bin/bash"})
	if err != nil {
		t.Fatalf("commandSpecFromRequest() error = %v", err)
	}

	if spec.Command != "/bin/sh" {
		t.Fatalf("command = %q, want %q", spec.Command, "/bin/sh")
	}
	if !reflect.DeepEqual(spec.Args, []string{"-lc", "echo"}) {
		t.Fatalf("args = %#v, want %#v", spec.Args, []string{"-lc", "echo"})
	}
}

func TestCommandSpecFromRequest_UsesDefaultWhenQueryMissing(t *testing.T) {
	req := httptest.NewRequest("GET", "/ws", nil)
	def := commandSpec{Command: "/bin/bash", Args: []string{"-l"}}

	spec, err := commandSpecFromRequest(req, def)
	if err != nil {
		t.Fatalf("commandSpecFromRequest() error = %v", err)
	}

	if spec.Command != def.Command {
		t.Fatalf("command = %q, want %q", spec.Command, def.Command)
	}
	if !reflect.DeepEqual(spec.Args, def.Args) {
		t.Fatalf("args = %#v, want %#v", spec.Args, def.Args)
	}
}

func TestBuildListenAddress(t *testing.T) {
	tests := []struct {
		name    string
		host    string
		address string
		want    string
	}{
		{name: "port only", host: "0.0.0.0", address: "8080", want: "0.0.0.0:8080"},
		{name: "with colon", host: "0.0.0.0", address: ":8080", want: ":8080"},
		{name: "full address", host: "0.0.0.0", address: "127.0.0.1:9000", want: "127.0.0.1:9000"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := buildListenAddress(tt.host, tt.address)
			if got != tt.want {
				t.Fatalf("buildListenAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}
