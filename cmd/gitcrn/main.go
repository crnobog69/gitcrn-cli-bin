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
	defaultServerURL = "http://100.91.132.35:5000"
	projectURL       = "https://github.com/crnobog69/gitcrn-cli-bin"
	creatorNames     = "crnijada / crnobog / vltc"
	latestReleaseAPI = "https://api.github.com/repos/crnobog69/gitcrn-cli-bin/releases/latest"
	updateLinuxCmd   = "curl -fsSL https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/update.sh | bash"
	updateWinCmd     = "iwr https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/update.ps1 -UseBasicParsing | iex"
	defaultCommitMsg = "❄️"

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

type appConfig struct {
	ServerURL string
	Token     string
	SSHAlias  string
	SSHHost   string
	SSHPort   int
	SSHUser   string
}

type giteaUser struct {
	Login string `json:"login"`
}

type giteaCreateRepoRequest struct {
	Name          string `json:"name"`
	Description   string `json:"description,omitempty"`
	Private       bool   `json:"private"`
	DefaultBranch string `json:"default_branch,omitempty"`
}

func main() {
	if len(os.Args) < 2 {
		if shouldCheckUpdates("") {
			maybePrintUpdateNotice()
		}
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
	case "completion":
		if err := runCompletion(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "-gc", "gc":
		if err := runGenerateConfig(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "-pp":
		if err := runMake([]string{"--push", "--pull"}, false); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "doctor":
		if err := runDoctor(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "make":
		if err := runMake(args, false); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "remake":
		if err := runMake(args, true); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "init":
		if err := runInit(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "generate":
		if err := runGenerate(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "create":
		if err := runCreate(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "repo":
		if err := runRepo(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "clone":
		if err := runClone(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "push":
		if err := runPush(args); err != nil {
			printError(err)
			os.Exit(1)
		}
	case "pull":
		if err := runPull(args); err != nil {
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

func runGenerate(args []string) error {
	if len(args) < 1 {
		printGenerateUsage(os.Stderr)
		return errors.New("generate тражи подкоманду")
	}

	switch args[0] {
	case "config":
		return runGenerateConfig(args[1:])
	case "-h", "--help", "help":
		printGenerateUsage(os.Stdout)
		return nil
	default:
		printGenerateUsage(os.Stderr)
		return fmt.Errorf("неподржана generate подкоманда: %s", args[0])
	}
}

func runGenerateConfig(args []string) error {
	fs := flag.NewFlagSet("generate config", flag.ContinueOnError)
	fs.SetOutput(io.Discard)
	force := fs.Bool("force", false, "Препиши постојећи config.toml")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printGenerateConfigUsage(os.Stdout)
			return nil
		}
		printGenerateConfigUsage(os.Stderr)
		return err
	}

	if fs.NArg() != 0 {
		printGenerateConfigUsage(os.Stderr)
		return fmt.Errorf("неочекивани аргументи: %s", strings.Join(fs.Args(), " "))
	}

	configPath, err := appConfigPath()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(configPath), 0o700); err != nil {
		return fmt.Errorf("креирање config директоријума: %w", err)
	}

	if !*force {
		if _, err := os.Stat(configPath); err == nil {
			return fmt.Errorf("%s већ постоји. Користи --force ако желиш препис", configPath)
		}
	}

	content := strings.Join([]string{
		"# gitcrn config",
		fmt.Sprintf("server_url = %q", defaultServerURL),
		"token = \"\"",
		"",
		fmt.Sprintf("ssh_alias = %q", defaultHostAlias),
		fmt.Sprintf("ssh_host = %q", defaultHostName),
		fmt.Sprintf("ssh_port = %d", defaultHostPort),
		fmt.Sprintf("ssh_user = %q", defaultHostUser),
		"",
	}, "\n")

	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		return fmt.Errorf("упис %s: %w", configPath, err)
	}

	fmt.Println(colorize("Креиран config: "+configPath, ansiGreen, stdoutColor))
	fmt.Println("Упиши token у config или постави env: GITCRN_TOKEN")
	return nil
}

func runCreate(args []string) error {
	if len(args) < 1 {
		printCreateUsage(os.Stderr)
		return errors.New("create тражи подкоманду")
	}

	switch args[0] {
	case "repo":
		return runCreateRepo(args[1:])
	case "-h", "--help", "help":
		printCreateUsage(os.Stdout)
		return nil
	default:
		printCreateUsage(os.Stderr)
		return fmt.Errorf("неподржана create подкоманда: %s", args[0])
	}
}

func runRepo(args []string) error {
	if len(args) < 1 {
		printRepoUsage(os.Stdout)
		return nil
	}

	switch args[0] {
	case "create":
		return runCreateRepo(args[1:])
	case "-h", "--help", "help":
		printRepoUsage(os.Stdout)
		return nil
	default:
		printRepoUsage(os.Stderr)
		return fmt.Errorf("неподржана repo подкоманда: %s", args[0])
	}
}

func runCreateRepo(args []string) error {
	fs := flag.NewFlagSet("create repo", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	private := fs.Bool("private", true, "Креирај private репозиторијум")
	public := fs.Bool("public", false, "Креирај public репозиторијум")
	desc := fs.String("desc", "", "Опис репозиторијума")
	defaultBranch := fs.String("default-branch", "", "Подразумевана грана (нпр main)")
	cloneNow := fs.Bool("clone", false, "Одмах клонирај после креирања")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printCreateRepoUsage(os.Stdout)
			return nil
		}
		printCreateRepoUsage(os.Stderr)
		return err
	}
	if fs.NArg() != 1 {
		printCreateRepoUsage(os.Stderr)
		return errors.New("create repo тражи owner/repo")
	}
	if *public {
		*private = false
	}

	owner, repoName, err := parseOwnerRepo(fs.Arg(0))
	if err != nil {
		return err
	}

	cfg, err := loadAppConfig()
	if err != nil {
		return err
	}
	token := strings.TrimSpace(os.Getenv("GITCRN_TOKEN"))
	if token == "" {
		token = strings.TrimSpace(os.Getenv("GITEA_TOKEN"))
	}
	if token == "" {
		token = strings.TrimSpace(cfg.Token)
	}
	if token == "" {
		return errors.New("недостаје token. Постави GITCRN_TOKEN или token у ~/.config/gitcrn/config.toml")
	}

	serverURL := strings.TrimSpace(cfg.ServerURL)
	if serverURL == "" {
		serverURL = defaultServerURL
	}
	serverURL = strings.TrimRight(serverURL, "/")

	login, err := giteaCurrentUser(serverURL, token)
	if err != nil {
		return fmt.Errorf("не могу да прочитам корисника преко API: %w", err)
	}

	endpoint := serverURL + "/api/v1/user/repos"
	if owner != login {
		endpoint = serverURL + "/api/v1/orgs/" + owner + "/repos"
	}

	payload := giteaCreateRepoRequest{
		Name:          repoName,
		Description:   strings.TrimSpace(*desc),
		Private:       *private,
		DefaultBranch: strings.TrimSpace(*defaultBranch),
	}

	if err := giteaCreateRepo(endpoint, token, payload); err != nil {
		return err
	}

	ownerRepo := fmt.Sprintf("%s/%s", owner, repoName)
	fmt.Println(colorize("Репозиторијум креиран: "+owner+"/"+repoName, ansiGreen, stdoutColor))

	if *cloneNow {
		return runClone([]string{ownerRepo})
	}
	fmt.Printf("Следеће: %s clone %s\n", appName, ownerRepo)
	fmt.Printf("У постојећем репоу: %s add %s\n", appName, ownerRepo)
	return nil
}

func parseOwnerRepo(input string) (owner, repo string, err error) {
	s := strings.TrimSpace(input)
	s = strings.TrimSuffix(strings.TrimPrefix(s, "/"), ".git")
	parts := strings.Split(s, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", errors.New("формат мора бити owner/repo")
	}
	return parts[0], parts[1], nil
}

func giteaCurrentUser(serverURL, token string) (string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, serverURL+"/api/v1/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", appName)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		msg, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("status %d: %s", resp.StatusCode, strings.TrimSpace(string(msg)))
	}

	var u giteaUser
	if err := json.NewDecoder(resp.Body).Decode(&u); err != nil {
		return "", err
	}
	if strings.TrimSpace(u.Login) == "" {
		return "", errors.New("празан login у API одговору")
	}
	return strings.TrimSpace(u.Login), nil
}

func giteaCreateRepo(endpoint, token string, payload giteaCreateRepoRequest) error {
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(string(body)))
	if err != nil {
		return err
	}
	req.Header.Set("Authorization", "token "+token)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", appName)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return nil
	}

	msg, _ := io.ReadAll(resp.Body)
	trimmed := strings.TrimSpace(string(msg))
	var errPayload map[string]any
	if err := json.Unmarshal(msg, &errPayload); err == nil {
		if v, ok := errPayload["message"].(string); ok && strings.TrimSpace(v) != "" {
			trimmed = strings.TrimSpace(v)
		}
	}
	if trimmed == "" {
		trimmed = "непозната грешка"
	}
	if resp.StatusCode == http.StatusConflict {
		return fmt.Errorf("repo већ постоји: %s", trimmed)
	}
	return fmt.Errorf("create repo неуспешан (status %d): %s", resp.StatusCode, trimmed)
}

func appConfigPath() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("детекција home директоријума: %w", err)
	}
	return filepath.Join(home, ".config", "gitcrn", "config.toml"), nil
}

func loadAppConfig() (appConfig, error) {
	cfg := appConfig{
		ServerURL: defaultServerURL,
		SSHAlias:  defaultHostAlias,
		SSHHost:   defaultHostName,
		SSHPort:   defaultHostPort,
		SSHUser:   defaultHostUser,
	}

	path, err := appConfigPath()
	if err != nil {
		return cfg, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return cfg, nil
		}
		return cfg, fmt.Errorf("читање %s: %w", path, err)
	}

	lines := strings.Split(normalizeNewlines(string(data)), "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if i := strings.Index(line, "#"); i >= 0 {
			line = strings.TrimSpace(line[:i])
		}
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.ToLower(strings.TrimSpace(parts[0]))
		val := strings.TrimSpace(parts[1])
		val = strings.Trim(strings.TrimSpace(val), "\"")

		switch key {
		case "server_url":
			if val != "" {
				cfg.ServerURL = val
			}
		case "token":
			cfg.Token = val
		case "ssh_alias":
			if val != "" {
				cfg.SSHAlias = val
			}
		case "ssh_host":
			if val != "" {
				cfg.SSHHost = val
			}
		case "ssh_user":
			if val != "" {
				cfg.SSHUser = val
			}
		case "ssh_port":
			if n, err := strconv.Atoi(val); err == nil && n > 0 && n <= 65535 {
				cfg.SSHPort = n
			}
		}
	}
	return cfg, nil
}

func runMake(args []string, overwrite bool) error {
	if len(args) > 0 && args[0] == "repo" {
		if overwrite {
			return errors.New("remake repo није подржан. Користи: gitcrn make repo owner/repo")
		}
		return runCreateRepo(args[1:])
	}

	fs := flag.NewFlagSet("make", flag.ContinueOnError)
	fs.SetOutput(io.Discard)

	makePush := fs.Bool("push", false, "Направи push скрипту")
	makePull := fs.Bool("pull", false, "Направи pull скрипту")
	makeBoth := fs.Bool("pp", false, "Краћи облик за --push --pull")

	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			printMakeUsage(os.Stdout)
			return nil
		}
		printMakeUsage(os.Stderr)
		return err
	}

	if fs.NArg() != 0 {
		printMakeUsage(os.Stderr)
		return fmt.Errorf("неочекивани аргументи: %s", strings.Join(fs.Args(), " "))
	}

	if *makeBoth {
		*makePush = true
		*makePull = true
	}
	if !*makePush && !*makePull {
		printMakeUsage(os.Stderr)
		return errors.New("изабери бар једно: --push, --pull или -pp")
	}

	if strings.TrimSpace(commandOutput("git", "rev-parse", "--is-inside-work-tree")) != "true" {
		return errors.New("ова команда мора да се покрене унутар git репозиторијума")
	}

	remoteOutput := commandOutput("git", "remote", "-v")
	if strings.TrimSpace(remoteOutput) == "" {
		doctorWarn("git remote -v", "није пронађен ниједан remote")
	} else {
		fmt.Println(colorize("Пронађени remote-и (git remote -v):", ansiCyan, stdoutColor))
		fmt.Println(remoteOutput)
	}

	fetchRemotes, pushRemotes := parseRemoteNames(remoteOutput)
	defaultBranch := strings.TrimSpace(commandOutput("git", "branch", "--show-current"))
	branch, err := promptInput(os.Stdout, os.Stdin, "Грана", defaultBranch)
	if err != nil {
		return fmt.Errorf("читање гране: %w", err)
	}

	created := make([]string, 0, 2)

	if *makePush {
		commitMsg, err := promptInput(os.Stdout, os.Stdin, "Порука за commit", defaultCommitMsg)
		if err != nil {
			return fmt.Errorf("читање commit поруке: %w", err)
		}

		defPushRemotes := strings.Join(preferNonEmpty(pushRemotes, fetchRemotes), ",")
		remotesText, err := promptInput(os.Stdout, os.Stdin, "Remote-и за push (зарез или размак)", defPushRemotes)
		if err != nil {
			return fmt.Errorf("читање push remote-а: %w", err)
		}
		remotes := parseRemoteList(remotesText)
		if len(remotes) == 0 {
			return errors.New("push захтева бар један remote")
		}

		path, err := writePushScript(commitMsg, branch, remotes, overwrite)
		if err != nil {
			return err
		}
		created = append(created, path)
	}

	if *makePull {
		defPullRemotes := strings.Join(preferNonEmpty(fetchRemotes, pushRemotes), ",")
		remotesText, err := promptInput(os.Stdout, os.Stdin, "Remote-и за pull (зарез или размак)", defPullRemotes)
		if err != nil {
			return fmt.Errorf("читање pull remote-а: %w", err)
		}
		remotes := parseRemoteList(remotesText)
		if len(remotes) == 0 {
			return errors.New("pull захтева бар један remote")
		}

		path, err := writePullScript(branch, remotes, overwrite)
		if err != nil {
			return err
		}
		created = append(created, path)
	}

	for _, p := range created {
		fmt.Println(colorize("Креирано: "+p, ansiGreen, stdoutColor))
	}
	return nil
}

func promptInput(w io.Writer, r io.Reader, label, defaultValue string) (string, error) {
	if strings.TrimSpace(defaultValue) != "" {
		fmt.Fprintf(w, "%s [%s]: ", label, defaultValue)
	} else {
		fmt.Fprintf(w, "%s: ", label)
	}

	reader := bufio.NewReader(r)
	line, err := reader.ReadString('\n')
	if err != nil && !errors.Is(err, io.EOF) {
		return "", err
	}

	value := strings.TrimSpace(line)
	if value == "" {
		return defaultValue, nil
	}
	return value, nil
}

func parseRemoteNames(remoteOutput string) (fetch, push []string) {
	fetchSeen := map[string]bool{}
	pushSeen := map[string]bool{}

	lines := strings.Split(strings.TrimSpace(remoteOutput), "\n")
	for _, line := range lines {
		fields := strings.Fields(strings.TrimSpace(line))
		if len(fields) < 3 {
			continue
		}
		name := fields[0]
		kind := strings.Trim(fields[len(fields)-1], "()")
		switch kind {
		case "fetch":
			if !fetchSeen[name] {
				fetchSeen[name] = true
				fetch = append(fetch, name)
			}
		case "push":
			if !pushSeen[name] {
				pushSeen[name] = true
				push = append(push, name)
			}
		}
	}
	return fetch, push
}

func parseRemoteList(input string) []string {
	seen := map[string]bool{}
	out := []string{}

	for _, raw := range strings.FieldsFunc(input, func(r rune) bool {
		return r == ',' || r == ' ' || r == '\t' || r == '\n'
	}) {
		item := strings.TrimSpace(raw)
		if item == "" || seen[item] {
			continue
		}
		seen[item] = true
		out = append(out, item)
	}
	return out
}

func preferNonEmpty(primary, fallback []string) []string {
	if len(primary) > 0 {
		return primary
	}
	return fallback
}

func writePushScript(commitMsg, branch string, remotes []string, overwrite bool) (string, error) {
	if runtime.GOOS == "windows" {
		path := "push.ps1"
		content := renderPushPS1(commitMsg, branch, remotes)
		if err := writeScriptFile(path, content, overwrite, 0o644); err != nil {
			return "", err
		}
		return path, nil
	}

	path := "push.sh"
	content := renderPushSh(commitMsg, branch, remotes)
	if err := writeScriptFile(path, content, overwrite, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func writePullScript(branch string, remotes []string, overwrite bool) (string, error) {
	if runtime.GOOS == "windows" {
		path := "pull.ps1"
		content := renderPullPS1(branch, remotes)
		if err := writeScriptFile(path, content, overwrite, 0o644); err != nil {
			return "", err
		}
		return path, nil
	}

	path := "pull.sh"
	content := renderPullSh(branch, remotes)
	if err := writeScriptFile(path, content, overwrite, 0o755); err != nil {
		return "", err
	}
	return path, nil
}

func writeScriptFile(path, content string, overwrite bool, mode os.FileMode) error {
	if !overwrite {
		if _, err := os.Stat(path); err == nil {
			return fmt.Errorf("%s већ постоји. Користи: gitcrn remake", path)
		}
	}

	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		return fmt.Errorf("не могу да упишем %s: %w", path, err)
	}
	return nil
}

func renderPushSh(commitMsg, branch string, remotes []string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString("COMMIT_MSG=" + shellSingleQuote(commitMsg) + "\n")
	sb.WriteString("BRANCH=" + shellSingleQuote(branch) + "\n")
	sb.WriteString("REMOTES=(")
	for i, r := range remotes {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(shellSingleQuote(r))
	}
	sb.WriteString(")\n\n")
	sb.WriteString("git add .\n")
	sb.WriteString("if git diff --cached --quiet; then\n")
	sb.WriteString("  echo \"Nema izmena za commit. Preskacem commit.\"\n")
	sb.WriteString("else\n")
	sb.WriteString("  git commit -m \"$COMMIT_MSG\"\n")
	sb.WriteString("fi\n\n")
	sb.WriteString("for remote in \"${REMOTES[@]}\"; do\n")
	sb.WriteString("  if [[ -n \"$BRANCH\" ]]; then\n")
	sb.WriteString("    git push \"$remote\" \"$BRANCH\"\n")
	sb.WriteString("  else\n")
	sb.WriteString("    git push \"$remote\"\n")
	sb.WriteString("  fi\n")
	sb.WriteString("done\n")
	return sb.String()
}

func renderPullSh(branch string, remotes []string) string {
	var sb strings.Builder
	sb.WriteString("#!/usr/bin/env bash\n")
	sb.WriteString("set -euo pipefail\n\n")
	sb.WriteString("BRANCH=" + shellSingleQuote(branch) + "\n")
	sb.WriteString("REMOTES=(")
	for i, r := range remotes {
		if i > 0 {
			sb.WriteString(" ")
		}
		sb.WriteString(shellSingleQuote(r))
	}
	sb.WriteString(")\n\n")
	sb.WriteString("for remote in \"${REMOTES[@]}\"; do\n")
	sb.WriteString("  if [[ -n \"$BRANCH\" ]]; then\n")
	sb.WriteString("    git pull \"$remote\" \"$BRANCH\"\n")
	sb.WriteString("  else\n")
	sb.WriteString("    git pull \"$remote\"\n")
	sb.WriteString("  fi\n")
	sb.WriteString("done\n")
	return sb.String()
}

func renderPushPS1(commitMsg, branch string, remotes []string) string {
	var sb strings.Builder
	sb.WriteString("$ErrorActionPreference = \"Stop\"\n")
	sb.WriteString("$CommitMessage = " + psSingleQuote(commitMsg) + "\n")
	sb.WriteString("$Branch = " + psSingleQuote(branch) + "\n")
	sb.WriteString("$Remotes = @(")
	for i, r := range remotes {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(psSingleQuote(r))
	}
	sb.WriteString(")\n\n")
	sb.WriteString("git add .\n")
	sb.WriteString("git diff --cached --quiet\n")
	sb.WriteString("if ($LASTEXITCODE -eq 0) {\n")
	sb.WriteString("  Write-Host \"Nema izmena za commit. Preskacem commit.\"\n")
	sb.WriteString("} else {\n")
	sb.WriteString("  git commit -m $CommitMessage\n")
	sb.WriteString("}\n\n")
	sb.WriteString("foreach ($remote in $Remotes) {\n")
	sb.WriteString("  if ([string]::IsNullOrWhiteSpace($Branch)) {\n")
	sb.WriteString("    git push $remote\n")
	sb.WriteString("  } else {\n")
	sb.WriteString("    git push $remote $Branch\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	return sb.String()
}

func renderPullPS1(branch string, remotes []string) string {
	var sb strings.Builder
	sb.WriteString("$ErrorActionPreference = \"Stop\"\n")
	sb.WriteString("$Branch = " + psSingleQuote(branch) + "\n")
	sb.WriteString("$Remotes = @(")
	for i, r := range remotes {
		if i > 0 {
			sb.WriteString(", ")
		}
		sb.WriteString(psSingleQuote(r))
	}
	sb.WriteString(")\n\n")
	sb.WriteString("foreach ($remote in $Remotes) {\n")
	sb.WriteString("  if ([string]::IsNullOrWhiteSpace($Branch)) {\n")
	sb.WriteString("    git pull $remote\n")
	sb.WriteString("  } else {\n")
	sb.WriteString("    git pull $remote $Branch\n")
	sb.WriteString("  }\n")
	sb.WriteString("}\n")
	return sb.String()
}

func shellSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

func psSingleQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", "''") + "'"
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

func runPush(args []string) error {
	if len(args) != 0 {
		printPushUsage(os.Stderr)
		return errors.New("push не прима додатне аргументе")
	}
	return runGeneratedScript("push")
}

func runPull(args []string) error {
	if len(args) != 0 {
		printPullUsage(os.Stderr)
		return errors.New("pull не прима додатне аргументе")
	}
	return runGeneratedScript("pull")
}

func runCompletion(args []string) error {
	if len(args) != 1 {
		printCompletionUsage(os.Stderr)
		return errors.New("completion тражи један аргумент: zsh, bash или fish")
	}

	script, err := completionScript(args[0])
	if err != nil {
		printCompletionUsage(os.Stderr)
		return err
	}
	fmt.Print(script)
	return nil
}

func completionScript(shell string) (string, error) {
	switch strings.ToLower(strings.TrimSpace(shell)) {
	case "zsh":
		return fmt.Sprintf(`#compdef %s

_%s() {
  local -a commands
  commands=(
    'generate:Генериши подешавања'
    'create:Креирај ресурсе'
    'repo:Repo namespace команде'
    'doctor:Провера окружења'
    'make:Генериши push/pull скрипте или make repo'
    'remake:Препиши push/pull скрипте'
    'init:Подеси SSH alias %s'
    'clone:Клонирај owner/repo преко SSH'
    'push:Покрени push.sh/push.ps1'
    'pull:Покрени pull.sh/pull.ps1'
    'add:Додај remote %s'
    'completion:Генериши shell completion'
    '-gc:Краћи облик за generate config'
    '-pp:Краћи облик за make --push --pull'
    '-v:Прикажи верзију'
    '--version:Прикажи верзију'
    'help:Помоћ'
  )

  local -a root_flags
  root_flags=(
    '-h[Помоћ]'
    '--help[Помоћ]'
  )

  local curcontext="$curcontext" state line
  _arguments -C \
    $root_flags \
    '1:команда:->cmds' \
    '*::аргумент:->args'

  case "$state" in
    cmds)
      _describe 'команде' commands
      ;;
    args)
      case "$line[1]" in
        init)
          _arguments '--default[Подразумевана SSH подешавања]' '--custom[Прилагођена SSH подешавања]' '--host[SSH HostName]:host:' '--port[SSH порт]:port:' '--user[SSH корисник]:user:'
          ;;
        completion)
          _values 'shell' zsh bash fish
          ;;
        generate)
          _values 'подкоманда' config
          ;;
        create)
          case "$line[2]" in
            repo)
              _arguments '--private[Креирај private репозиторијум]' '--public[Креирај public репозиторијум]' '--desc[Опис]:опис:' '--default-branch[Грана]:грана:' '--clone[Одмах клонирај]'
              ;;
            *)
              _values 'подкоманда' repo
              ;;
          esac
          ;;
        repo)
          case "$line[2]" in
            create)
              _arguments '--private[Креирај private репозиторијум]' '--public[Креирај public репозиторијум]' '--desc[Опис]:опис:' '--default-branch[Грана]:грана:' '--clone[Одмах клонирај]'
              ;;
            *)
              _values 'подкоманда' create
              ;;
          esac
          ;;
        make)
          _arguments '1:подкоманда/опција:(repo --push --pull -pp)' '*::аргумент:->makeargs'
          case "$line[2]" in
            repo)
              _arguments '--private[Креирај private репозиторијум]' '--public[Креирај public репозиторијум]' '--desc[Опис]:опис:' '--default-branch[Грана]:грана:' '--clone[Одмах клонирај]'
              ;;
          esac
          ;;
        remake)
          _arguments '--push[Генериши push скрипту]' '--pull[Генериши pull скрипту]' '-pp[И push и pull]'
          ;;
        clone|add)
          _message 'owner/repo'
          ;;
      esac
      ;;
  esac
}

_%s "$@"
`, appName, appName, defaultHostAlias, defaultHostAlias, appName), nil
	case "bash":
		return fmt.Sprintf(`_%s_complete() {
  local cur prev words cword
  cur="${COMP_WORDS[COMP_CWORD]}"
  prev=""
  if [[ $COMP_CWORD -gt 0 ]]; then
    prev="${COMP_WORDS[COMP_CWORD-1]}"
  fi
  words=("${COMP_WORDS[@]}")
  cword=$COMP_CWORD

  local root_cmds="generate create repo doctor make remake init clone push pull add completion -gc -pp -v --version help"
  local opts="-h --help"

  if [[ $cword -eq 1 ]]; then
    COMPREPLY=( $(compgen -W "$root_cmds $opts" -- "$cur") )
    return
  fi

  case "${words[1]}" in
    init)
      COMPREPLY=( $(compgen -W "--default --custom --host --port --user -h --help" -- "$cur") )
      ;;
    completion)
      COMPREPLY=( $(compgen -W "zsh bash fish" -- "$cur") )
      ;;
    generate)
      COMPREPLY=( $(compgen -W "config -h --help" -- "$cur") )
      ;;
    create)
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "repo -h --help" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "--private --public --desc --default-branch --clone -h --help" -- "$cur") )
      fi
      ;;
    repo)
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "create -h --help" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "--private --public --desc --default-branch --clone -h --help" -- "$cur") )
      fi
      ;;
    make)
      if [[ $cword -eq 2 ]]; then
        COMPREPLY=( $(compgen -W "repo --push --pull -pp -h --help" -- "$cur") )
      elif [[ "${words[2]}" == "repo" ]]; then
        COMPREPLY=( $(compgen -W "--private --public --desc --default-branch --clone -h --help" -- "$cur") )
      else
        COMPREPLY=( $(compgen -W "--push --pull -pp -h --help" -- "$cur") )
      fi
      ;;
    remake)
      COMPREPLY=( $(compgen -W "--push --pull -pp -h --help" -- "$cur") )
      ;;
    clone|add)
      COMPREPLY=()
      ;;
  esac
}

complete -F _%s_complete %s
`, appName, appName, appName), nil
	case "fish":
		return fmt.Sprintf(`complete -c %s -f
complete -c %s -n "__fish_use_subcommand" -a "generate create repo doctor make remake init clone push pull add completion -gc -pp -v --version help"
complete -c %s -n "__fish_seen_subcommand_from completion" -a "zsh bash fish"
complete -c %s -n "__fish_seen_subcommand_from generate" -a "config"
complete -c %s -n "__fish_seen_subcommand_from create" -a "repo"
complete -c %s -n "__fish_seen_subcommand_from repo" -a "create"
complete -c %s -n "__fish_seen_subcommand_from make" -a "repo"
complete -c %s -n "__fish_seen_subcommand_from create; and __fish_seen_subcommand_from repo" -l private
complete -c %s -n "__fish_seen_subcommand_from create; and __fish_seen_subcommand_from repo" -l public
complete -c %s -n "__fish_seen_subcommand_from create; and __fish_seen_subcommand_from repo" -l desc -r
complete -c %s -n "__fish_seen_subcommand_from create; and __fish_seen_subcommand_from repo" -l default-branch -r
complete -c %s -n "__fish_seen_subcommand_from create; and __fish_seen_subcommand_from repo" -l clone
complete -c %s -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from create" -l private
complete -c %s -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from create" -l public
complete -c %s -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from create" -l desc -r
complete -c %s -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from create" -l default-branch -r
complete -c %s -n "__fish_seen_subcommand_from repo; and __fish_seen_subcommand_from create" -l clone
complete -c %s -n "__fish_seen_subcommand_from make; and __fish_seen_subcommand_from repo" -l private
complete -c %s -n "__fish_seen_subcommand_from make; and __fish_seen_subcommand_from repo" -l public
complete -c %s -n "__fish_seen_subcommand_from make; and __fish_seen_subcommand_from repo" -l desc -r
complete -c %s -n "__fish_seen_subcommand_from make; and __fish_seen_subcommand_from repo" -l default-branch -r
complete -c %s -n "__fish_seen_subcommand_from make; and __fish_seen_subcommand_from repo" -l clone
complete -c %s -n "__fish_seen_subcommand_from make remake" -l push
complete -c %s -n "__fish_seen_subcommand_from make remake" -l pull
complete -c %s -n "__fish_seen_subcommand_from make remake" -o pp
complete -c %s -n "__fish_seen_subcommand_from init" -l default
complete -c %s -n "__fish_seen_subcommand_from init" -l custom
complete -c %s -n "__fish_seen_subcommand_from init" -l host -r
complete -c %s -n "__fish_seen_subcommand_from init" -l port -r
complete -c %s -n "__fish_seen_subcommand_from init" -l user -r
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName), nil
	default:
		return "", fmt.Errorf("неподржан shell: %s (подржано: zsh, bash, fish)", shell)
	}
}

func runGeneratedScript(kind string) error {
	var path string
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		path = kind + ".ps1"
		if !fileExists(path) {
			return fmt.Errorf("%s није пронађен. Покрени: gitcrn make -pp", path)
		}
		cmd = exec.Command("powershell", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", path)
	default:
		path = kind + ".sh"
		if !fileExists(path) {
			return fmt.Errorf("%s није пронађен. Покрени: gitcrn make -pp", path)
		}
		cmd = exec.Command("bash", path)
	}

	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("извршавање %s није успело: %w", path, err)
	}
	return nil
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
	case "completion":
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
  %s generate config
  %s -gc
  %s completion zsh|bash|fish
  %s create repo owner/repo
  %s make repo owner/repo
  %s repo create owner/repo
  %s doctor
  %s make --push --pull
  %s remake -pp
  %s -pp
  %s init --default
  %s init --custom --host <host> --port <port> --user <user>
  %s clone owner/repo [directory]
  %s push
  %s pull
  %s add owner/repo
  %s -v | --version

Примери:
  %s generate config
  %s -gc --force
  %s completion zsh > ~/.zsh/completions/_gitcrn
  %s create repo vltc/mojrepo --private --clone
  %s make repo vltc/mojrepo --private --clone
  %s repo create crnbg/platform --public
  %s doctor
  %s make --push --pull
  %s remake --push
  %s -pp
  %s init --default
  %s init --custom --host 100.91.132.35 --port 222 --user git
  %s clone vltc/kapri
  %s push
  %s pull
  %s add vltc/crnbg
`, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName, appName)
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

func printPushUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s push
`, appName)
}

func printPullUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s pull
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

func printGenerateUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s generate config
  %s -gc
`, appName, appName)
}

func printGenerateConfigUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s generate config [--force]
  %s -gc [--force]
`, appName, appName)
}

func printCompletionUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s completion zsh
  %s completion bash
  %s completion fish
`, appName, appName, appName)
}

func printCreateUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s create repo owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
  %s make repo owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
`, appName, appName)
}

func printRepoUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s repo create owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
`, appName)
}

func printCreateRepoUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s create repo owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
  %s make repo owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
  %s repo create owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
`, appName, appName, appName)
}

func printMakeUsage(w io.Writer) {
	fmt.Fprintf(w, `Коришћење:
  %s make repo owner/repo [--private|--public] [--desc "..."] [--default-branch main] [--clone]
  %s make --push --pull
  %s make -pp
  %s remake --push --pull
  %s remake -pp
  %s -pp
`, appName, appName, appName, appName, appName, appName)
}
