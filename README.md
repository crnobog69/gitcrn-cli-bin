# gitcrn-cli

Приватни CLI алат за Gitea сервер `gitcrn`.

## Шта ради

- Подешава SSH `Host gitcrn` у `~/.ssh/config`
- Генерише конфиг: `gitcrn generate config` или `gitcrn -gc`
- Креира репо преко Gitea API: `gitcrn create repo owner/repo`
- Клонира репо: `gitcrn clone owner/repo`
- Додаје remote `gitcrn`: `gitcrn add owner/repo`
- Проверава окружење: `gitcrn doctor`
- Прави `push`/`pull` скрипте у тренутном репоу: `gitcrn make` / `gitcrn remake`
- Покреће генерисане скрипте: `gitcrn push` / `gitcrn pull`
- При покретању проверава да ли постоји нова верзија и исписује команду за ажурирање

## Важно

- За `gitcrn init` мораш да имаш инсталиран **Tailscale**
- Ако већ постоји иста SSH конфигурација, алат неће преписивати фајл

## Инсталација (Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/install.sh | bash
```

## Инсталација (Windows)

```powershell
iwr https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/install.ps1 -UseBasicParsing | iex
```

## Ажурирање (Linux)

```bash
curl -fsSL https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/update.sh | bash
```

## Ажурирање (Windows)

```powershell
iwr https://raw.githubusercontent.com/crnobog69/gitcrn-cli-bin/refs/heads/master/scripts/update.ps1 -UseBasicParsing | iex
```

## Коришћење

```bash
gitcrn doctor
gitcrn -gc
gitcrn create repo vltc/mojrepo --private --clone
gitcrn init --default
gitcrn clone vltc/kapri
gitcrn add vltc/crnbg
gitcrn make --push --pull
gitcrn push
gitcrn pull
gitcrn -pp
gitcrn -v
```

## Shell completion

`gitcrn` има уграђену команду за completion:

```bash
gitcrn completion zsh
gitcrn completion bash
gitcrn completion fish
```

### Zsh

```bash
mkdir -p ~/.zsh/completions
gitcrn completion zsh > ~/.zsh/completions/_gitcrn
echo 'fpath=(~/.zsh/completions $fpath)' >> ~/.zshrc
echo 'autoload -Uz compinit && compinit' >> ~/.zshrc
source ~/.zshrc
```

### Bash

```bash
mkdir -p ~/.local/share/bash-completion/completions
gitcrn completion bash > ~/.local/share/bash-completion/completions/gitcrn
```

### Fish

```bash
mkdir -p ~/.config/fish/completions
gitcrn completion fish > ~/.config/fish/completions/gitcrn.fish
```

## Конфиг и token

- Конфиг фајл: `~/.config/gitcrn/config.toml`
- Генерисање шаблона:
  - `gitcrn generate config`
  - `gitcrn -gc`
- Token се чита редом:
  - `GITCRN_TOKEN` (или `GITEA_TOKEN`)
  - `~/.config/gitcrn/config.toml` (`token = "..."`)

## `create repo`

- Главна команда: `gitcrn create repo owner/repo`
- Alias: `gitcrn repo create owner/repo`
- Опције:
  - `--private` (default)
  - `--public`
  - `--desc "..."`
  - `--default-branch main`
  - `--clone`

## `make` / `remake`

- `gitcrn make --push --pull` прави скрипте (`push.sh`/`pull.sh` на Linux-у, `push.ps1`/`pull.ps1` на Windows-у)
- `gitcrn -pp` је пречица за `make --push --pull`
- `gitcrn remake ...` ради исто, али преписује постојеће скрипте
- При креирању:
  - чита `git remote -v`
  - пита за грану
  - пита за remote-е за push/pull
  - пита за commit поруку (ако притиснеш Enter, подразумевано је `❄`)
- После креирања можеш да радиш:
  - `gitcrn push`
  - `gitcrn pull`

## `doctor` шта проверава

- Tailscale верзију (или да ли недостаје)
- Git и `user.name` / `user.email` (локално и глобално)
- `Host gitcrn` подешавање у SSH конфигу
- Који SSH public key је пронађен и његов коментар (обично име/мејл)

## Провера нове верзије

- На сваком покретању (осим `--version`/`--help`) проверава latest release на GitHub-у
- Ако постоји новија, само испише команду за ажурирање
- Нема аутоматског ажурирања
- Искључивање провере: `GITCRN_NO_UPDATE_CHECK=1 gitcrn ...`

## Прилагођени `init`

```bash
gitcrn init --custom --host 100.91.132.35 --port 222 --user git
```

## SSH блок који `init --default` прави

```sshconfig
Host gitcrn
    HostName 100.91.132.35
    User git
    Port 222
```

## Примери

```bash
# Клонирање
gitcrn clone vltc/kapri

# Додавање remote-а у постојећем репоу
gitcrn add vltc/crnbg
```

## За одржавање изворног кода (maintainer)

```bash
./scripts/release.sh --version v0.1.0
```

```bash
rem run -D RELEASE_VERSION=v0.5.0 github-release
```

`RELEASE_VERSION` мора бити production tag у формату `vX.Y.Z` (нпр `v0.5.0`).

То генерише:

- `dist/gitcrn-linux-amd64`
- `dist/gitcrn-linux-arm64`
- `dist/gitcrn-windows-amd64.exe`
- `dist/gitcrn-windows-arm64.exe`
- `dist/checksums.txt`
