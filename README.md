# Aman

Music management tools.

Support flac, mp3, m4a (aac).

## Installation

Require go 1.24+ and `ffprobe` present on `$PATH`.

If you are using conversion features, you will also need `ffmpeg` present on `$PATH`.

```shell
go install github.com/mawngo/aman@latest
```

# Usage

### Organize

Organize music files into folder by bitrate

- `flac` lossless
- `320aac` >= 320kbps
- `256aac` >= 256kbps
- `320mp3` >= 320kbps
- `128mp3` >= 128kbps
- `low` < 128kbps

```shell
> aman sort .\mydir
```

| Before        | After           |
|---------------|-----------------|
| 128.mp3       | /128mp3/128.mp3 |
| 320.mp3       | /320mp3/320.mp3 |
| 320.m4a       | /320aac/320.m4a |
| 96.mp3        | /low/96.mp3     |
| lossless.flac | /lossless.flac  |

By default, flac files will be kept in the root directory.
You can change this behavior using `-g` flag.

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
> aman -h  
Music management tools (.flac, .mp3, .m4a)

Usage:
  aman [command]

Available Commands:
  ls          List audio files
  meta        Print metadata of audio files
  sort        Organize audio files to groups by bitrate
  copy        Selectively copy music files by bitrate group
  chk         Checking missing audio files by bitrate group
  conv        Fill missing audio bitrate groups by converting from FLAC
  clean       Remove audio of bitrate groups if it already exists in other quality
  help        Help about any command
  completion  Generate the autocompletion script for the specified shell

Flags:
  -h, --help         help for aman
      --log string   Configure log level [default/verbose/quiet] (default "default")

Use "aman [command] --help" for more information about a command.
```
