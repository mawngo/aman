package audio

import (
	"aman/internal/fileutils"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type ConvertConfig struct {
	Src          string
	TargetDir    string
	TargetGroups []string
	Flat         bool
	Overwrite    bool
	DryRun       bool
}

func Convert(conf ConvertConfig) int64 {
	if len(conf.TargetGroups) == 0 {
		return 0
	}
	if conf.Flat && len(conf.TargetGroups) > 1 {
		slog.Error("Flat mode is not supported with multiple target groups")
		return 0
	}

	targetDir := conf.TargetDir
	if targetDir == "" {
		targetDir = filepath.Dir(conf.Src)
	}
	targetDir = ungroupDir(targetDir)
	location := LocationDir(conf.Src)
	basename := extractBasename(conf.Src)
	count := int64(0)

	for _, group := range conf.TargetGroups {
		var ext, bitrate string
		if bitrate = strings.TrimSuffix(group, TypeAAC); bitrate != group {
			ext = "m4a"
			bitrate += "k"
		} else if bitrate = strings.TrimSuffix(group, TypeMp3); bitrate != group {
			ext = "mp3"
			bitrate += "k"
		} else {
			slog.Error("Output format not supported",
				slog.String("basename", conf.Src),
				slog.String("group", group))
			continue
		}

		dest := filepath.Join(targetDir, group, basename+"."+ext)
		if conf.Flat {
			dest = filepath.Join(targetDir, basename+"."+ext)
			println(dest)
		}
		dir := filepath.Dir(dest)
		if !conf.Overwrite {
			if _, err := os.Stat(dest); err == nil {
				slog.Warn("File already exists", slog.String("file", dest))
				continue
			} else if !os.IsNotExist(err) {
				slog.Error("Error converting",
					slog.String("basename", basename),
					slog.String("group", group),
					slog.String("from", location),
					slog.String("to", dir),
					slog.Any("err", err))
				continue
			}
		}

		slog.Info("Converting",
			slog.String("basename", basename),
			slog.String("group", group),
			slog.String("from", location),
			slog.String("to", dir))
		if !conf.DryRun {
			var cmd *exec.Cmd
			if ext == "mp3" {
				cmd = exec.Command("ffmpeg",
					"-v", "error",
					"-i", conf.Src,
					"-ab", bitrate,
					"-c:v", "copy",
					"-map_metadata", "0",
					"-id3v2_version", "3",
					dest)
			} else {
				cmd = exec.Command("ffmpeg",
					"-v", "error",
					"-i", conf.Src,
					"-c:a", "aac",
					"-b:a", bitrate,
					"-c:v", "copy",
					"-map_metadata", "0",
					dest)
			}

			fileutils.EnsureDir(dir)
			if err := cmd.Run(); err != nil {
				slog.Error("Error converting",
					slog.String("basename", basename),
					slog.String("group", group),
					slog.String("from", location),
					slog.String("to", dir),
					slog.Any("err", err))
				continue
			}
			count++
		}

		slog.Info("Converted",
			slog.String("basename", basename),
			slog.String("group", group),
			slog.String("from", location),
			slog.String("to", dir))
	}
	return count
}
