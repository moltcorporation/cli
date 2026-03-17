package cmd

import (
	"reflect"
	"testing"
)

func TestSanitizeMetaArgs(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
		want []string
	}{
		{
			name: "root version drops profile flag and value",
			args: []string{"moltcorp", "--version", "--profile", "builder"},
			want: []string{"moltcorp", "--version"},
		},
		{
			name: "explicit version drops profile flag and value",
			args: []string{"moltcorp", "version", "--profile", "builder"},
			want: []string{"moltcorp", "version"},
		},
		{
			name: "subcommand help drops auth flags",
			args: []string{"moltcorp", "agents", "--help", "--profile", "builder", "--api-key", "secret"},
			want: []string{"moltcorp", "agents", "--help"},
		},
		{
			name: "normal command is unchanged",
			args: []string{"moltcorp", "--profile", "builder", "agents", "me"},
			want: []string{"moltcorp", "--profile", "builder", "agents", "me"},
		},
		{
			name: "git push is unchanged",
			args: []string{"moltcorp", "git", "push", "--profile", "builder", "origin", "main"},
			want: []string{"moltcorp", "git", "push", "--profile", "builder", "origin", "main"},
		},
		{
			name: "non-meta version flag after command is unchanged",
			args: []string{"moltcorp", "context", "--version", "--profile", "builder"},
			want: []string{"moltcorp", "context", "--version", "--profile", "builder"},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := sanitizeMetaArgs(tt.args)
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("sanitizeMetaArgs() = %#v, want %#v", got, tt.want)
			}
		})
	}
}

func TestExtractMoltcorpFlags(t *testing.T) {
	t.Parallel()

	apiKey, profile, baseURL, gitArgs := extractMoltcorpFlags([]string{
		"--profile", "builder",
		"--api-key=secret",
		"origin",
		"main",
		"--base-url", "https://example.com",
		"--dry-run",
	})

	if apiKey != "secret" {
		t.Fatalf("apiKey = %q, want %q", apiKey, "secret")
	}
	if profile != "builder" {
		t.Fatalf("profile = %q, want %q", profile, "builder")
	}
	if baseURL != "https://example.com" {
		t.Fatalf("baseURL = %q, want %q", baseURL, "https://example.com")
	}

	wantArgs := []string{"origin", "main", "--dry-run"}
	if !reflect.DeepEqual(gitArgs, wantArgs) {
		t.Fatalf("gitArgs = %#v, want %#v", gitArgs, wantArgs)
	}
}
