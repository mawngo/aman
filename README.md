# Aman

Music management tools.

## Installation

Require go 1.24+ and `ffprobe` present on `$PATH`.

```shell
go install github.com/mawngo/aman@latest
```

# Usage

### Organize

Organize music files into folder by bitrate

- `320mp3` >= 320kbps
- `128mp3` < 320kbps
- `flac` lossless

```shell
> aman .\mydir
```

By default, files with bitrate >= 320kbps will be kept in the root directory.
You can change this behavior using `-g` flag.

| Before        | After               |
|---------------|---------------------|
| 128.mp3       | /128mp3/128.mp3     |
| 320.mp3       | 320.mp3             |
| 500.mp3       | 500.mp3             |
| 96.mp3        | /128mp3/96.mp3      |
| lossless.flac | /flac/lossless.flac |

### Copy

Selectively copy music files from one directory to another, based on their bitrate.

```shell
> aman .\mydir .\targetdir
```

| mydir         | targetdir |
|---------------|-----------|
| /128mp3/a.mp3 | -         |
| /128mp3/c.mp3 | c.mp3     |
| /320mp3/a.mp3 | a.mp3     |
| /320mp3/b.mp3 | -         |
| /flac/b.flac  | b.flac    |

### Options

```
> kcomp -h  
Music management tools (.flac, .mp3)

Usage:
  aman [command]

Available Commands:
  meta        Print metadata of audio file
  sort        Organize audio files by bitrate
  copy        Selectively copy music files by bitrate
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

Flags:
  -h, --help         help for aman
      --log string   Configure log level [default/verbose/quiet] (default "default")

Use "aman [command] --help" for more information about a command.

```
