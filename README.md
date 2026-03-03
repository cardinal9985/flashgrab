```
 ▛▀▘▌  ▞▀▖▞▀▖▌ ▌▞▀▖▛▀▖▞▀▖▛▀▖
 ▙▄ ▌  ▙▄▌▚▄ ▙▄▌▌▄▖▙▄▘▙▄▌▙▄▘
 ▌  ▌  ▌ ▌▖ ▌▌ ▌▌ ▌▌▚ ▌ ▌▌ ▌
 ▘  ▀▀▘▘ ▘▝▀ ▘ ▘▝▀ ▘ ▘▘ ▘▀▀
```

**Grab Flash and browser games from the web.**

A terminal tool for downloading `.swf` files and other browser games from
Newgrounds, itch.io, Kongregate, and Internet Archive. Games are
cross-referenced against the [Flashpoint Archive](https://flashpointarchive.org)
database for accurate naming and metadata.

![demo](demo.gif)

## Features

- Interactive TUI with file selection and download progress
- CLI mode for scripting (`flashgrab <url>`)
- **Newgrounds**, **itch.io**, **Kongregate**, **Internet Archive** support
- Cross-references the [Flashpoint Archive](https://flashpointarchive.org) DB — matches by source URL and title, shows developer/platform info
- Skips already-downloaded files
- Atomic downloads (partial files cleaned up on failure)
- First-run setup wizard, re-accessible via `flashgrab config` or `ctrl+s`

## Install

```sh
# go
go install github.com/cardinal9985/flashgrab/cmd/flashgrab@latest

# homebrew
brew install cardinal9985/tap/flashgrab

# aur
yay -S flashgrab-bin

# nix
nix run github:cardinal9985/flashgrab

# debian/ubuntu — grab .deb from releases
sudo dpkg -i flashgrab_*.deb
```

Or download a binary from [releases](https://github.com/cardinal9985/flashgrab/releases).

## Usage

```sh
flashgrab                                              # interactive TUI
flashgrab https://www.newgrounds.com/portal/view/59593 # direct download
flashgrab config                                       # re-run setup
```

First run will ask for a download directory and an optional itch.io API key
(get one at https://itch.io/user/settings/api-keys). Config is stored at
`~/.config/flashgrab/config.toml`.

## Supported sites

| Site | Downloads | Auth |
|------|-----------|------|
| [Newgrounds](https://newgrounds.com) | SWF, MP4, WebM | No |
| [itch.io](https://itch.io) | Any upload (SWF, ZIP, HTML5, etc.) | API key |
| [Kongregate](https://kongregate.com) | SWF | No |
| [Internet Archive](https://archive.org) | SWF, ZIP, HTML | No |

## Flashpoint cross-referencing

When a game is resolved, flashgrab queries the Flashpoint Archive API to find a
matching entry. It scores results by source URL similarity first, then falls
back to title matching. A confirmed match renames the file to the canonical
Flashpoint title and displays metadata (developer, platform) on the done screen.

## Building

```sh
git clone https://github.com/cardinal9985/flashgrab.git
cd flashgrab
make build   # or: go build ./cmd/flashgrab
make test    # runs go test ./...
```

Requires Go 1.24+.

## License

[MIT](LICENSE)
