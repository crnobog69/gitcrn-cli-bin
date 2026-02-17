package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	appName          = "gitcrn"
	defaultHostAlias = "gitcrn"
	defaultHostName  = "100.91.132.35"
	defaultHostUser  = "git"
	defaultHostPort  = 222
)

var version = "dev"

func main() {
	if len(os.Args) < 2 {
		printRootUsage(os.Stderr)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	switch cmd {
	case "-v", "--version", "version":
		fmt.Printf("%s %s\n", appName, version)
	case "init":
		if err := runInit(args); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "clone":
		if err := runClone(args); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "remote":
		if err := runRemote(args); err != nil {
			fmt.Fprintln(os.Stderr, "Error:", err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printRootUsage(os.Stdout)
	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n\n", cmd)
		printRootUsage(os.Stderr)
		os.Exit(1)
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	defaultMode := fs.Bool("default", false, "Use the default private server settings")
	customMode := fs.Bool("custom", false, "Use custom server settings")
	host := fs.String("host", "", "Custom SSH HostName")
	port := fs.Int("port", 0, "Custom SSH Port")
	user := fs.String("user", "", "Custom SSH User")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printInitUsage(os.Stdout)
			return nil
		}
		printInitUsage(os.Stderr)
		return err
	}

	if *defaultMode == *customMode {
		printInitUsage(os.Stderr)
		return errors.New("choose exactly one mode: --default or --custom")
	}

	if fs.NArg() != 0 {
		printInitUsage(os.Stderr)
		return fmt.Errorf("unexpected arguments: %s", strings.Join(fs.Args(), " "))
	}

	finalHost := defaultHostName
	finalPort := defaultHostPort
	finalUser := defaultHostUser

	if *customMode {
		if strings.TrimSpace(*host) == "" {
			return errors.New("--host is required with --custom")
		}
		if strings.TrimSpace(*user) == "" {
			return errors.New("--user is required with --custom")
		}
		if *port <= 0 || *port > 65535 {
			return errors.New("--port must be between 1 and 65535 with --custom")
		}

		finalHost = strings.TrimSpace(*host)
		finalPort = *port
		finalUser = strings.TrimSpace(*user)
	}

	configPath, err := upsertSSHConfig(defaultHostAlias, finalHost, finalUser, finalPort)
	if err != nil {
		return err
	}

	fmt.Printf("Updated SSH config: %s\n", configPath)
	fmt.Printf("Host %s -> %s:%d as %s\n", defaultHostAlias, finalHost, finalPort, finalUser)
	return nil
}

func runClone(args []string) error {
	if len(args) < 1 {
		printCloneUsage(os.Stderr)
		return errors.New("clone requires owner/repo")
	}

	repoURL, err := buildRepoURL(args[0])
	if err != nil {
		return err
	}

	gitArgs := []string{"clone", repoURL}
	if len(args) > 1 {
		gitArgs = append(gitArgs, args[1:]...)
	}

	return runGit(gitArgs...)
}

func runRemote(args []string) error {
	if len(args) < 1 {
		printRemoteUsage(os.Stderr)
		return errors.New("remote requires a subcommand")
	}

	switch args[0] {
	case "add":
		return runRemoteAdd(args[1:])
	case "-h", "--help", "help":
		printRemoteUsage(os.Stdout)
		return nil
	default:
		printRemoteUsage(os.Stderr)
		return fmt.Errorf("unsupported remote subcommand: %s", args[0])
	}
}

func runRemoteAdd(args []string) error {
	if len(args) != 2 {
		printRemoteAddUsage(os.Stderr)
		return errors.New("usage: gitcrn remote add <name> owner/repo")
	}

	name := strings.TrimSpace(args[0])
	if name == "" {
		return errors.New("remote name cannot be empty")
	}

	repoURL, err := buildRepoURL(args[1])
	if err != nil {
		return err
	}

	return runGit("remote", "add", name, repoURL)
}

func buildRepoURL(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", errors.New("repository cannot be empty")
	}

	if strings.Contains(s, "://") {
		return "", errors.New("use owner/repo format, not a full URL")
	}

	if strings.HasPrefix(s, defaultHostAlias+":") {
		s = strings.TrimPrefix(s, defaultHostAlias+":")
	}

	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimPrefix(s, "/")

	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("repository must be in owner/repo format")
	}

	return fmt.Sprintf("%s:%s/%s.git", defaultHostAlias, parts[0], parts[1]), nil
}

func upsertSSHConfig(alias, host, user string, port int) (string, error) {
	configPath, err := sshConfigPath()
	if err != nil {
		return "", err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return "", fmt.Errorf("create .ssh directory: %w", err)
	}

	content := ""
	data, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", fmt.Errorf("read %s: %w", configPath, err)
	}
	if err == nil {
		content = normalizeNewlines(string(data))
	}

	block := renderSSHHostBlock(alias, host, user, port)
	updated := mergeSSHHostBlock(content, alias, block)

	if err := os.WriteFile(configPath, []byte(updated), 0o600); err != nil {
		return "", fmt.Errorf("write %s: %w", configPath, err)
	}

	return configPath, nil
}

func sshConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("detect home directory: %w", err)
	}
	return filepath.Join(home, ".ssh", "config"), nil
}

func renderSSHHostBlock(alias, host, user string, port int) string {
	return strings.Join([]string{
		fmt.Sprintf("Host %s", alias),
		fmt.Sprintf("    HostName %s", host),
		fmt.Sprintf("    User %s", user),
		fmt.Sprintf("    Port %d", port),
	}, "\n")
}

func mergeSSHHostBlock(content, alias, block string) string {
	trimmed := strings.TrimSpace(content)
	if trimmed == "" {
		return block + "\n"
	}

	lines := strings.Split(content, "\n")
	var out []string
	replaced := false

	for i := 0; i < len(lines); {
		curr := strings.TrimSpace(lines[i])
		if patterns, ok := parseHostLine(curr); ok && hostPatternMatches(patterns, alias) {
			if !replaced {
				if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
					out = append(out, "")
				}
				out = append(out, strings.Split(block, "\n")...)
				replaced = true
			}

			i++
			for i < len(lines) {
				next := strings.TrimSpace(lines[i])
				if _, nextIsHost := parseHostLine(next); nextIsHost {
					break
				}
				i++
			}
			continue
		}

		out = append(out, lines[i])
		i++
	}

	if !replaced {
		if len(out) > 0 && strings.TrimSpace(out[len(out)-1]) != "" {
			out = append(out, "")
		}
		out = append(out, strings.Split(block, "\n")...)
	}

	result := strings.TrimRight(strings.Join(out, "\n"), "\n")
	return result + "\n"
}

func parseHostLine(line string) ([]string, bool) {
	fields := strings.Fields(line)
	if len(fields) < 2 || !strings.EqualFold(fields[0], "Host") {
		return nil, false
	}
	return fields[1:], true
}

func hostPatternMatches(patterns []string, alias string) bool {
	for _, p := range patterns {
		if strings.EqualFold(strings.TrimSpace(p), alias) {
			return true
		}
	}
	return false
}

func normalizeNewlines(s string) string {
	return strings.ReplaceAll(s, "\r\n", "\n")
}

func runGit(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("git %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}

func printRootUsage(w io.Writer) {
	fmt.Fprintf(w, `%s - private Gitea helper CLI

Usage:
  %s init --default
  %s init --custom --host <host> --port <port> --user <user>
  %s clone owner/repo [directory]
  %s remote add <name> owner/repo
  %s -v | --version

Examples:
  %s init --default
  %s init --custom --host 100.91.132.35 --port 222 --user git
  %s clone vltc/kapri
  %s remote add gitcrn vltc/crnbg
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}

func printInitUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  %s init --default
  %s init --custom --host <host> --port <port> --user <user>
`, appName, appName)
}

func printCloneUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  %s clone owner/repo [directory]
`, appName)
}

func printRemoteUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  %s remote add <name> owner/repo
`, appName)
}

func printRemoteAddUsage(w io.Writer) {
	fmt.Fprintf(w, `Usage:
  %s remote add <name> owner/repo
`, appName)
}
