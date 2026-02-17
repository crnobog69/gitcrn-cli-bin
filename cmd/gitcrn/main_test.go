package main

import (
	"strings"
	"testing"
)

func TestBuildRepoURL(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		want    string
		wantErr bool
	}{
		{
			name:  "plain owner repo",
			input: "vltc/kapri",
			want:  "gitcrn:vltc/kapri.git",
		},
		{
			name:  "accept .git suffix",
			input: "vltc/kapri.git",
			want:  "gitcrn:vltc/kapri.git",
		},
		{
			name:  "accept prefixed alias",
			input: "gitcrn:vltc/kapri.git",
			want:  "gitcrn:vltc/kapri.git",
		},
		{
			name:    "reject empty",
			input:   "",
			wantErr: true,
		},
		{
			name:    "reject https",
			input:   "https://example.com/vltc/kapri.git",
			wantErr: true,
		},
		{
			name:    "reject missing repo",
			input:   "vltc",
			wantErr: true,
		},
		{
			name:    "reject extra segments",
			input:   "group/vltc/kapri",
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got, err := buildRepoURL(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("expected error, got none (value: %q)", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %q, want %q", got, tc.want)
			}
		})
	}
}

func TestMergeSSHHostBlockReplaceAndAppend(t *testing.T) {
	block := renderSSHHostBlock("gitcrn", "100.91.132.35", "git", 222)

	existing := strings.Join([]string{
		"Host github.com",
		"    HostName github.com",
		"",
		"Host gitcrn",
		"    HostName old.example",
		"    User old",
		"    Port 2022",
		"",
		"Host internal",
		"    HostName 10.0.0.5",
		"",
	}, "\n")

	got := mergeSSHHostBlock(existing, "gitcrn", block)
	if strings.Contains(got, "old.example") {
		t.Fatalf("old gitcrn host block should be replaced")
	}
	if !strings.Contains(got, "HostName 100.91.132.35") {
		t.Fatalf("updated block missing expected HostName: %q", got)
	}

	withoutTarget := strings.Join([]string{
		"Host github.com",
		"    HostName github.com",
		"",
	}, "\n")
	appended := mergeSSHHostBlock(withoutTarget, "gitcrn", block)
	if !strings.Contains(appended, "Host gitcrn") {
		t.Fatalf("expected gitcrn block to be appended: %q", appended)
	}
}
