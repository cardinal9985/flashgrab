```
 _____ _           _     ____           _
|  ___| | __ _ ___| |__ / ___|_ __ __ _| |__
| |_  | |/ _` / __| '_ \| |  _| '__/ _` | '_ \
|  _| | | (_| \__ \ | | | |_| | | | (_| | |_) |
|_|   |_|\__,_|___/_| |_|\____|_|  \__,_|_.__/
```

**Grab Flash and browser games from the web.**

A terminal UI for downloading `.swf` files and other browser games from
Newgrounds, itch.io, Kongregate, and Internet Archive. Built for game
preservation, ROM managers, and anyone who wants to save Flash games before
they disappear.

![demo](demo.gif)

## Features

- Interactive TUI with search, file selection, and download progress
- Non-interactive CLI mode for scripting (`flashgrab <url>`)
- Downloads from **Newgrounds**, **itch.io**, **Kongregate**, and **Internet Archive**
- Automatic filename matching against the [Flashpoint Archive](https://flashpointarchive.org) database
- Skips files that already exist (no duplicates)
- Atomic downloads — partial files are cleaned up on failure
- Configurable download directory via `~/.config/flashgrab/config.toml`
- API keys stored with restrictive file permissions

## Install

### Go

```sh
go install github.com/cardinal9985/flashgrab/cmd/flashgrab@latest
```

### Homebrew

```sh
brew install cardinal9985/tap/flashgrab
```

### Arch Linux (AUR)

```sh
yay -S flashgrab-bin
```

### Nix

```sh
nix run github:cardinal9985/flashgrab
```

Or add to your flake inputs and include `flashgrab.packages.${system}.default`
in your packages.

### Debian / Ubuntu

Download the `.deb` from the [releases page](https://github.com/cardinal9985/flashgrab/releases):

```sh
sudo dpkg -i flashgrab_*.deb
```

### Binary

Grab a binary from [releases](https://github.com/cardinal9985/flashgrab/releases)
and put it somewhere on your `$PATH`.

## Usage

### TUI mode

```sh
flashgrab
```

Launches the interactive interface. Paste a URL, pick your files, watch them
download. Press `n` after a download to grab another, `q` to quit.

### CLI mode

```sh
flashgrab https://www.newgrounds.com/portal/view/218014
```

Downloads directly to your configured directory. No TUI, no fuss. Useful in
scripts or paired with `xargs`.

### First-time setup

On first run, flashgrab asks for:

1. **Download directory** — where files get saved (default: `~/Downloads`)
2. **itch.io API key** — optional, only needed for itch.io downloads

Get an itch.io key at https://itch.io/user/settings/api-keys

Re-run the setup wizard anytime:

```sh
flashgrab config
```

Config lives at `~/.config/flashgrab/config.toml`.

## Supported sites

| Site | What it grabs | Auth needed |
|------|---------------|-------------|
| [Newgrounds](https://newgrounds.com) | SWF, MP4, WebM from portal pages | No |
| [itch.io](https://itch.io) | Any uploaded file (SWF, ZIP, HTML5, etc.) | API key |
| [Kongregate](https://kongregate.com) | SWF files from game pages | No |
| [Internet Archive](https://archive.org) | SWF, ZIP, HTML from item pages or direct links | No |

Filenames are cross-referenced against the [Flashpoint Archive](https://flashpointarchive.org)
database when possible, so your collection stays consistent with the
preservation community's naming.

## Building from source

```sh
git clone https://github.com/cardinal9985/flashgrab.git
cd flashgrab
make build
```

Requires Go 1.21+.

## License

[MIT](LICENSE)
