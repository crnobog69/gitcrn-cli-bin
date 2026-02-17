# gitcrn-cli

Приватни CLI алат за Gitea сервер `gitcrn`.

## Шта ради

- Подешава SSH `Host gitcrn` у `~/.ssh/config`
- Клонира репо: `gitcrn clone owner/repo`
- Додаје remote `gitcrn`: `gitcrn add owner/repo`
- Проверава окружење: `gitcrn doctor`
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
gitcrn init --default
gitcrn clone vltc/kapri
gitcrn add vltc/crnbg
gitcrn -v
```

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

То генерише:

- `dist/gitcrn-linux-amd64`
- `dist/gitcrn-linux-arm64`
- `dist/gitcrn-windows-amd64.exe`
- `dist/gitcrn-windows-arm64.exe`
- `dist/checksums.txt`
