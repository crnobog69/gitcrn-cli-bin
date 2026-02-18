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

func TestHasExactSSHHostConfig(t *testing.T) {
	content := strings.Join([]string{
		"Host gitcrn",
		"    HostName 100.91.132.35",
		"    User git",
		"    Port 222",
		"",
	}, "\n")

	if !hasExactSSHHostConfig(content, "gitcrn", "100.91.132.35", "git", 222) {
		t.Fatalf("expected exact ssh config to match")
	}

	if hasExactSSHHostConfig(content, "gitcrn", "100.91.132.35", "git", 220) {
		t.Fatalf("unexpected match for wrong port")
	}
}

func TestCompareSemver(t *testing.T) {
	tests := []struct {
		a      string
		b      string
		want   int
		ok     bool
		testID string
	}{
		{a: "v0.2.0", b: "v0.1.9", want: 1, ok: true, testID: "newer"},
		{a: "0.1.0", b: "v0.1.0", want: 0, ok: true, testID: "same"},
		{a: "v1.0.0", b: "dev", want: 0, ok: false, testID: "non-semver-local"},
		{a: "v0.1.0", b: "v0.2.0", want: -1, ok: true, testID: "older"},
	}

	for _, tc := range tests {
		got, ok := compareSemver(tc.a, tc.b)
		if ok != tc.ok {
			t.Fatalf("%s: ok=%v want %v", tc.testID, ok, tc.ok)
		}
		if ok && got != tc.want {
			t.Fatalf("%s: got=%d want=%d", tc.testID, got, tc.want)
		}
	}
}

func TestParseRemoteNames(t *testing.T) {
	out := strings.Join([]string{
		"origin\thttps://example.com/repo.git (fetch)",
		"origin\thttps://example.com/repo.git (push)",
		"gitcrn\tgitcrn:vltc/repo.git (fetch)",
		"gitcrn\tgitcrn:vltc/repo.git (push)",
		"backup\tgit@backup/repo.git (push)",
	}, "\n")

	fetch, push := parseRemoteNames(out)
	if strings.Join(fetch, ",") != "origin,gitcrn" {
		t.Fatalf("unexpected fetch remotes: %v", fetch)
	}
	if strings.Join(push, ",") != "origin,gitcrn,backup" {
		t.Fatalf("unexpected push remotes: %v", push)
	}
}

func TestParseRemoteList(t *testing.T) {
	got := parseRemoteList("origin, gitcrn backup origin")
	if strings.Join(got, ",") != "origin,gitcrn,backup" {
		t.Fatalf("unexpected list: %v", got)
	}
}

func TestParseOwnerRepo(t *testing.T) {
	owner, repo, err := parseOwnerRepo("vltc/kapri")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if owner != "vltc" || repo != "kapri" {
		t.Fatalf("unexpected owner/repo: %s/%s", owner, repo)
	}

	_, _, err = parseOwnerRepo("kapri")
	if err == nil {
		t.Fatalf("expected error for invalid repo spec")
	}
}

func TestCompletionScript(t *testing.T) {
	tests := []string{"zsh", "bash", "fish"}
	for _, shell := range tests {
		out, err := completionScript(shell)
		if err != nil {
			t.Fatalf("unexpected error for %s: %v", shell, err)
		}
		if strings.TrimSpace(out) == "" {
			t.Fatalf("expected non-empty completion output for %s", shell)
		}
		if !strings.Contains(out, "completion") {
			t.Fatalf("completion output for %s should mention completion command", shell)
		}
	}

	if _, err := completionScript("pwsh"); err == nil {
		t.Fatalf("expected unsupported shell error")
	}
}

func TestShouldCheckUpdates(t *testing.T) {
	t.Setenv("GITCRN_NO_UPDATE_CHECK", "")

	if shouldCheckUpdates("completion") {
		t.Fatalf("completion should not trigger update checks")
	}
	if shouldCheckUpdates("--version") {
		t.Fatalf("--version should not trigger update checks")
	}
	if !shouldCheckUpdates("doctor") {
		t.Fatalf("doctor should trigger update checks")
	}

	t.Setenv("GITCRN_NO_UPDATE_CHECK", "1")
	if shouldCheckUpdates("doctor") {
		t.Fatalf("env flag should disable update checks")
	}
}
