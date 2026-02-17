package main

import (
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"
)

const (
	appName          = "gitcrn"
	defaultHostAlias = "gitcrn"
	defaultHostName  = "100.91.132.35"
	defaultHostUser  = "git"
	defaultHostPort  = 222
	projectURL       = "https://github.com/crnobog69/gitcrn-cli-bin"
	creatorNames     = "cnrnijada / crnobog / vltc"
	latestReleaseAPI = "https://api.github.com/repos/crnobog69/gitcrn-cli-bin/releases/latest"
	updateLinuxCmd   = "curl -fsSL https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/update.sh | bash"
	updateWinCmd     = "iwr https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/update.ps1 -UseBasicParsing | iex"

	ansiReset  = "\033[0m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
)

var version = "dev"

var (
	stdoutColor = detectColor(os.Stdout)
	stderrColor = detectColor(os.Stderr)
)

func main() {
	if len(os.Args) < 2 {
		printRootUsage(os.Stderr)
		os.Exit(1)
	}

	cmd := os.Args[1]
	args := os.Args[2:]

	if shouldCheckUpdates(cmd) {
		maybePrintUpdateNotice()
	}

	switch cmd {
	case "-v", "--version", "version":
		printVersion(os.Stdout)
	case "doctor":
		if err := runDoctor(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "clone":
		if err := runClone(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "add":
		if err := runAdd(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "remote":
		// Legacy support: gitcrn remote add gitcrn owner/repo
		if err := runRemote(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "-h", "--help", "help":
		printRootUsage(os.Stdout)
	default:
		fmt.Fprintln(os.Stderr, colorize("Грешка: непозната команда: "+cmd, ansiRed, stderrColor))
		fmt.Fprintln(os.Stderr)
		printRootUsage(os.Stderr)
		os.Exit(1)
	}
}

func runInit(args []string) error {
	fs := flag.NewFlagSet("init", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	defaultMode := fs.Bool("default", false, "Користи подразумевана подешавања")
	customMode := fs.Bool("custom", false, "Користи прилагођена подешавања")
	host := fs.String("host", "", "SSH HostName")
	port := fs.Int("port", 0, "SSH Port")
	user := fs.String("user", "", "SSH User")

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
		return errors.New("изабери тачно један режим: --default или --custom")
	}

	if fs.NArg() != 0 {
		printInitUsage(os.Stderr)
		return fmt.Errorf("неочекивани аргументи: %s", strings.Join(fs.Args(), " "))
	}

	if err := ensureTailscaleAvailable(); err != nil {
		return err
	}

	finalHost := defaultHostName
	finalPort := defaultHostPort
	finalUser := defaultHostUser

	if *customMode {
		if strings.TrimSpace(*host) == "" {
			return errors.New("--host је обавезан уз --custom")
		}
		if strings.TrimSpace(*user) == "" {
			return errors.New("--user је обавезан уз --custom")
		}
		if *port <= 0 || *port > 65535 {
			return errors.New("--port мора бити између 1 и 65535 уз --custom")
		}

		finalHost = strings.TrimSpace(*host)
		finalPort = *port
		finalUser = strings.TrimSpace(*user)
	}

	configPath, changed, err := upsertSSHConfig(defaultHostAlias, finalHost, finalUser, finalPort)
	if err != nil {
		return err
	}

	if !changed {
		fmt.Println(colorize("SSH конфигурација већ постоји: "+configPath, ansiYellow, stdoutColor))
		fmt.Printf("Host %s -> %s:%d као %s\n", defaultHostAlias, finalHost, finalPort, finalUser)
		return nil
	}

	fmt.Println(colorize("SSH конфигурација ажурирана: "+configPath, ansiGreen, stdoutColor))
	fmt.Printf("Host %s -> %s:%d као %s\n", defaultHostAlias, finalHost, finalPort, finalUser)
	return nil
}

func runDoctor(args []string) error {
	fs := flag.NewFlagSet("doctor", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printDoctorUsage(os.Stdout)
			return nil
		}
		printDoctorUsage(os.Stderr)
		return err
	}

	if fs.NArg() != 0 {
		printDoctorUsage(os.Stderr)
		return fmt.Errorf("неочекивани аргументи: %s", strings.Join(fs.Args(), " "))
	}

	fmt.Println(colorize("Провера окружења (doctor)", ansiCyan, stdoutColor))
	doctorCheckTailscale()
	doctorCheckGitIdentity()
	doctorCheckSSHConfig()
	return nil
}

func doctorCheckTailscale() {
	out, err := exec.Command("tailscale", "version").CombinedOutput()
	if err != nil {
		doctorWarn("Tailscale", "није инсталиран или није у PATH-у")
		fmt.Printf("  Инсталација: https://tailscale.com/download\n")
		return
	}

	firstLine := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
	if firstLine == "" {
		firstLine = "доступан"
	}
	doctorOK("Tailscale", firstLine)
}

func doctorCheckGitIdentity() {
	if out, err := exec.Command("git", "--version").CombinedOutput(); err != nil {
		doctorWarn("Git", "није инсталиран или није у PATH-у")
		return
	} else {
		line := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
		if line == "" {
			line = "доступан"
		}
		doctorOK("Git", line)
	}

	nameGlobal := strings.TrimSpace(commandOutput("git", "config", "--global", "--get", "user.name"))
	emailGlobal := strings.TrimSpace(commandOutput("git", "config", "--global", "--get", "user.email"))
	nameLocal := strings.TrimSpace(commandOutput("git", "config", "--get", "user.name"))
	emailLocal := strings.TrimSpace(commandOutput("git", "config", "--get", "user.email"))

	if nameLocal != "" || emailLocal != "" {
		doctorOK("Git идентитет (локални)", fmt.Sprintf("%s <%s>", fallback(nameLocal, "?"), fallback(emailLocal, "?")))
	} else {
		doctorWarn("Git идентитет (локални)", "није подешен у тренутном репозиторијуму")
	}

	if nameGlobal != "" || emailGlobal != "" {
		doctorOK("Git идентитет (глобални)", fmt.Sprintf("%s <%s>", fallback(nameGlobal, "?"), fallback(emailGlobal, "?")))
	} else {
		doctorWarn("Git идентитет (глобални)", "постави: git config --global user.name \"Твоје Име\" && git config --global user.email \"ти@мејл\"")
	}
}

func doctorCheckSSHConfig() {
	configPath, err := sshConfigPath()
	if err != nil {
		doctorWarn("SSH конфиг", err.Error())
		return
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			doctorWarn("SSH конфиг", fmt.Sprintf("не постоји (%s). Покрени: gitcrn init --default", configPath))
			return
		}
		doctorWarn("SSH конфиг", fmt.Sprintf("грешка читања %s: %v", configPath, err))
		return
	}

	content := normalizeNewlines(string(data))
	settings, found := findSSHHostSettings(content, defaultHostAlias)
	if !found {
		doctorWarn("SSH host gitcrn", "није пронађен у ~/.ssh/config. Покрени: gitcrn init --default")
		return
	}

	hostName := fallback(settings["hostname"], "?")
	user := fallback(settings["user"], "?")
	port := fallback(settings["port"], "?")
	doctorOK("SSH host gitcrn", fmt.Sprintf("HostName=%s User=%s Port=%s", hostName, user, port))

	identity := strings.TrimSpace(settings["identityfile"])
	pubPath := ""
	if identity != "" {
		pubPath = identityPublicKeyPath(identity)
	}
	if pubPath == "" {
		pubPath = firstExistingPath(defaultPublicKeyCandidates()...)
	}

	if pubPath == "" {
		doctorWarn("SSH кључ", "ниједан .pub кључ није пронађен у ~/.ssh")
		return
	}

	keyType, comment, err := readPublicKeyInfo(pubPath)
	if err != nil {
		doctorWarn("SSH кључ", fmt.Sprintf("%s (не могу да прочитам коментар: %v)", pubPath, err))
		return
	}

	if comment == "" {
		comment = "(без коментара)"
	}
	doctorOK("SSH кључ", fmt.Sprintf("%s [%s] коментар: %s", pubPath, keyType, comment))
}

func doctorOK(name, details string) {
	fmt.Printf("%s %s: %s\n", colorize("[OK]", ansiGreen, stdoutColor), name, details)
}

func doctorWarn(name, details string) {
	fmt.Printf("%s %s: %s\n", colorize("[WARN]", ansiYellow, stdoutColor), name, details)
}

func runClone(args []string) error {
	if len(args) < 1 {
		printCloneUsage(os.Stderr)
		return errors.New("clone тражи owner/repo")
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

func runAdd(args []string) error {
	if len(args) != 1 {
		printAddUsage(os.Stderr)
		return errors.New("add тражи owner/repo")
	}

	repoURL, err := buildRepoURL(args[0])
	if err != nil {
		return err
	}

	return runGit("remote", "add", defaultHostAlias, repoURL)
}

func runRemote(args []string) error {
	if len(args) < 1 {
		printRemoteUsage(os.Stderr)
		return errors.New("remote тражи подкоманду")
	}

	switch args[0] {
	case "add":
		// Support: gitcrn remote add gitcrn owner/repo
		if len(args) == 3 && strings.TrimSpace(args[1]) == defaultHostAlias {
			return runAdd(args[2:])
		}
		printRemoteUsage(os.Stderr)
		return errors.New("legacy облик је: gitcrn remote add gitcrn owner/repo")
	case "-h", "--help", "help":
		printRemoteUsage(os.Stdout)
		return nil
	default:
		printRemoteUsage(os.Stderr)
		return fmt.Errorf("неподржана remote подкоманда: %s", args[0])
	}
}

func shouldCheckUpdates(cmd string) bool {
	if os.Getenv("GITCRN_NO_UPDATE_CHECK") != "" {
		return false
	}
	switch cmd {
	case "-h", "--help", "help", "-v", "--version", "version":
		return false
	default:
		return true
	}
}

func maybePrintUpdateNotice() {
	latestTag, err := fetchLatestReleaseTag(1200 * time.Millisecond)
	if err != nil {
		return
	}

	cmp, ok := compareSemver(latestTag, version)
	if !ok || cmp <= 0 {
		return
	}

	fmt.Println(colorize(fmt.Sprintf("Доступна је нова верзија: %s (тренутна %s)", latestTag, version), ansiYellow, stdoutColor))
	fmt.Println("Ажурирај овом командом:")
	if runtime.GOOS == "windows" {
		fmt.Printf("  %s\n", updateWinCmd)
		return
	}
	fmt.Printf("  %s\n", updateLinuxCmd)
}

func fetchLatestReleaseTag(timeout time.Duration) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, latestReleaseAPI, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", appName)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return "", fmt.Errorf("status %d", resp.StatusCode)
	}

	var payload struct {
		TagName string `json:"tag_name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.TagName) == "" {
		return "", errors.New("empty tag_name")
	}
	return strings.TrimSpace(payload.TagName), nil
}

func compareSemver(a, b string) (int, bool) {
	av, okA := parseVersionParts(a)
	bv, okB := parseVersionParts(b)
	if !okA || !okB {
		return 0, false
	}

	maxLen := len(av)
	if len(bv) > maxLen {
		maxLen = len(bv)
	}
	for i := 0; i < maxLen; i++ {
		ai := 0
		bi := 0
		if i < len(av) {
			ai = av[i]
		}
		if i < len(bv) {
			bi = bv[i]
		}
		if ai > bi {
			return 1, true
		}
		if ai < bi {
			return -1, true
		}
	}
	return 0, true
}

func parseVersionParts(v string) ([]int, bool) {
	s := strings.TrimSpace(v)
	s = strings.TrimPrefix(strings.TrimPrefix(s, "v"), "V")
	if s == "" {
		return nil, false
	}

	raw := strings.Split(s, ".")
	parts := make([]int, 0, len(raw))
	for _, seg := range raw {
		if seg == "" {
			return nil, false
		}
		n, ok := leadingInt(seg)
		if !ok {
			return nil, false
		}
		parts = append(parts, n)
	}
	return parts, true
}

func leadingInt(s string) (int, bool) {
	var b strings.Builder
	for _, r := range s {
		if r < '0' || r > '9' {
			break
		}
		b.WriteRune(r)
	}
	if b.Len() == 0 {
		return 0, false
	}
	n, err := strconv.Atoi(b.String())
	if err != nil {
		return 0, false
	}
	return n, true
}

func ensureTailscaleAvailable() error {
	cmd := exec.Command("tailscale", "version")
	out, err := cmd.CombinedOutput()
	if err == nil {
		firstLine := strings.TrimSpace(strings.SplitN(string(out), "\n", 2)[0])
		if firstLine != "" {
			fmt.Println(colorize("Tailscale: "+firstLine, ansiCyan, stdoutColor))
		}
		return nil
	}

	fmt.Fprintln(os.Stderr, colorize("Упозорење: Tailscale није доступан. Без Tailscale-а SSH ка gitcrn неће радити.", ansiYellow, stderrColor))

	install, askErr := promptYesNo(os.Stderr, os.Stdin, "Да ли желиш упутство за инсталацију Tailscale-а? [y/N]: ")
	if askErr == nil && install {
		printTailscaleInstallHint(os.Stderr)
	}

	return errors.New("инсталирај Tailscale па понови: gitcrn init --default")
}

func promptYesNo(w io.Writer, r io.Reader, prompt string) (bool, error) {
	fmt.Fprint(w, prompt)

	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return false, err
	}

	answer := strings.ToLower(strings.TrimSpace(line))
	switch answer {
	case "y", "yes", "d", "da", "д", "да":
		return true, nil
	default:
		return false, nil
	}
}

func printTailscaleInstallHint(w io.Writer) {
	fmt.Fprintln(w, "Инсталација Tailscale-а:")
	switch runtime.GOOS {
	case "linux":
		fmt.Fprintln(w, "  Linux: curl -fsSL https://tailscale.com/install.sh | sh")
	case "windows":
		fmt.Fprintln(w, "  Windows: https://tailscale.com/download/windows")
	default:
		fmt.Fprintln(w, "  Погледај: https://tailscale.com/download")
	}
}

func buildRepoURL(input string) (string, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return "", errors.New("repo не сме бити празан")
	}

	if strings.Contains(s, "://") {
		return "", errors.New("користи owner/repo формат, не пун URL")
	}

	if strings.HasPrefix(s, defaultHostAlias+":") {
		s = strings.TrimPrefix(s, defaultHostAlias+":")
	}

	s = strings.TrimSuffix(s, ".git")
	s = strings.TrimPrefix(s, "/")

	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", errors.New("repo мора бити у owner/repo формату")
	}

	return fmt.Sprintf("%s:%s/%s.git", defaultHostAlias, parts[0], parts[1]), nil
}

func upsertSSHConfig(alias, host, user string, port int) (string, bool, error) {
	configPath, err := sshConfigPath()
	if err != nil {
		return "", false, err
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return "", false, fmt.Errorf("креирање .ssh директоријума: %w", err)
	}

	content := ""
	data, err := os.ReadFile(configPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", false, fmt.Errorf("читање %s: %w", configPath, err)
	}
	if err == nil {
		content = normalizeNewlines(string(data))
	}

	if hasExactSSHHostConfig(content, alias, host, user, port) {
		return configPath, false, nil
	}

	block := renderSSHHostBlock(alias, host, user, port)
	updated := mergeSSHHostBlock(content, alias, block)

	if err := os.WriteFile(configPath, []byte(updated), 0o600); err != nil {
		return "", false, fmt.Errorf("упис %s: %w", configPath, err)
	}

	return configPath, true, nil
}

func hasExactSSHHostConfig(content, alias, host, user string, port int) bool {
	settings, found := findSSHHostSettings(content, alias)
	if !found {
		return false
	}

	hostNameOk := strings.EqualFold(strings.TrimSpace(settings["hostname"]), strings.TrimSpace(host))
	userOk := strings.EqualFold(strings.TrimSpace(settings["user"]), strings.TrimSpace(user))
	portOk := strings.TrimSpace(settings["port"]) == fmt.Sprintf("%d", port)

	return hostNameOk && userOk && portOk
}

func findSSHHostSettings(content, alias string) (map[string]string, bool) {
	lines := strings.Split(content, "\n")

	for i := 0; i < len(lines); i++ {
		patterns, isHostLine := parseHostLine(strings.TrimSpace(lines[i]))
		if !isHostLine || !hostPatternMatches(patterns, alias) {
			continue
		}

		settings := map[string]string{}
		for j := i + 1; j < len(lines); j++ {
			line := strings.TrimSpace(lines[j])
			if line == "" || strings.HasPrefix(line, "#") {
				continue
			}
			if _, nextHost := parseHostLine(line); nextHost {
				break
			}

			fields := strings.Fields(line)
			if len(fields) < 2 {
				continue
			}
			key := strings.ToLower(fields[0])
			value := strings.Join(fields[1:], " ")
			if _, exists := settings[key]; !exists {
				settings[key] = strings.TrimSpace(value)
			}
		}
		return settings, true
	}

	return nil, false
}

func sshConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("детекција home директоријума: %w", err)
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
		return fmt.Errorf("git %s није успео: %w", strings.Join(args, " "), err)
	}
	return nil
}

func commandOutput(name string, args ...string) string {
	out, err := exec.Command(name, args...).CombinedOutput()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func fallback(value, alt string) string {
	if strings.TrimSpace(value) == "" {
		return alt
	}
	return value
}

func defaultPublicKeyCandidates() []string {
	return []string{
		"~/.ssh/id_ed25519.pub",
		"~/.ssh/id_rsa.pub",
		"~/.ssh/id_ecdsa.pub",
		"~/.ssh/id_dsa.pub",
	}
}

func identityPublicKeyPath(identityFile string) string {
	id := strings.TrimSpace(identityFile)
	if id == "" {
		return ""
	}
	id = expandHomePath(id)
	id = strings.Trim(id, "\"")

	if strings.HasSuffix(id, ".pub") {
		if fileExists(id) {
			return id
		}
		return ""
	}

	pub := id + ".pub"
	if fileExists(pub) {
		return pub
	}
	return ""
}

func firstExistingPath(paths ...string) string {
	for _, p := range paths {
		pp := expandHomePath(strings.TrimSpace(p))
		if fileExists(pp) {
			return pp
		}
	}
	return ""
}

func fileExists(path string) bool {
	if path == "" {
		return false
	}
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

func expandHomePath(path string) string {
	p := strings.TrimSpace(path)
	if p == "" {
		return p
	}
	if p == "~" || strings.HasPrefix(p, "~/") {
		home, err := os.UserHomeDir()
		if err != nil {
			return p
		}
		if p == "~" {
			return home
		}
		return filepath.Join(home, strings.TrimPrefix(p, "~/"))
	}
	return p
}

func readPublicKeyInfo(path string) (keyType, comment string, err error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}

	line := strings.TrimSpace(strings.SplitN(string(data), "\n", 2)[0])
	fields := strings.Fields(line)
	if len(fields) < 2 {
		return "", "", fmt.Errorf("невалидан public key формат")
	}

	keyType = fields[0]
	if len(fields) > 2 {
		comment = strings.Join(fields[2:], " ")
	}
	return keyType, comment, nil
}

func printError(err error) {
	if err == nil {
		return
	}
	fmt.Fprintln(os.Stderr, colorize("Грешка: "+err.Error(), ansiRed, stderrColor))
}

func detectColor(file *os.File) bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	if strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	info, err := file.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0
}

func colorize(msg, ansi string, enabled bool) string {
	if !enabled {
		return msg
	}
	return ansi + msg + ansiReset
}

func printVersion(w io.Writer) {
	fmt.Fprintf(w, `%s %s
Направио: %s
Линк: %s
`, appName, version, creatorNames, projectURL)
}

func printRootUsage(w io.Writer) {
	fmt.Fprintf(w, `%s - приватни Gitea CLI алат

Коришћење:
  %s doctor
  %s init --default
  %s init --custom --host <host> --port <port> --user <user>
  %s clone owner/repo [directory]
  %s add owner/repo
  %s -v | --version

Примери:
  %s doctor
  %s init --default
  %s init --custom --host 100.91.132.35 --port 222 --user git
  %s clone vltc/kapri
  %s add vltc/crnbg
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
}

func printInitUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s init --default
  %s init --custom --host <host> --port <port> --user <user>
`, appName, appName)
}

func printCloneUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s clone owner/repo [directory]
`, appName)
}

func printAddUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s add owner/repo
`, appName)
}

func printRemoteUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s remote add gitcrn owner/repo
`, appName)
}

func printDoctorUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s doctor
`, appName)
}
