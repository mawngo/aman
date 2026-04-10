package audio

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/dustin/go-humanize"
	"log/slog"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	Group320aac = "320aac"
	Group256aac = "256aac"
	Group320Mp3 = "320mp3"
	Group128Mp3 = "128mp3"
	GroupFLAC   = "flac"

	TypeMp3  = "mp3"
	TypeFlac = "flac"
	TypeAAC  = "aac"

	LowQuality = "low"
)

// Groups is a map of audio group to quality level.
var Groups = map[string]int{
	GroupFLAC: 100,

	Group320aac: 5,
	Group320Mp3: 4,
	Group256aac: 3,

	Group128Mp3: 1,

	LowQuality: 0,
}

var ErrNotAudioFile = errors.New("not an audio file")
var ErrInvalidFormat = errors.New("invalid audio format")

func Probe(filename string) (ProbedAudio, error) {
	cmd := exec.Command("ffprobe",
		"-v", "error",
		"-of", "flat=s=_",
		"-select_streams", "a:0",
		"-show_entries", "stream=bit_rate,codec_name:format=duration,size:format_tags",
		"-print_format", "json",
		filename)
	output, err := cmd.Output()
	if err != nil {
		return ProbedAudio{}, err
	}
	var data ProbeResult
	err = json.Unmarshal(output, &data)
	if err != nil {
		return ProbedAudio{}, err
	}
	if len(data.Streams) == 0 {
		return ProbedAudio{}, ErrNotAudioFile
	}

	duration, err := strconv.ParseFloat(data.Format.Duration, 64)
	if err != nil {
		return ProbedAudio{}, fmt.Errorf("invalid duration \"%s\": %w", data.Format.Duration, ErrInvalidFormat)
	}
	size, err := strconv.ParseUint(data.Format.Size, 10, 64)
	if err != nil {
		return ProbedAudio{}, fmt.Errorf("invalid size \"%s\": %w", data.Format.Size, ErrInvalidFormat)
	}

	bitrate := uint64(0)
	if data.Streams[0].BitRate != "" {
		bitrate, err = strconv.ParseUint(data.Streams[0].BitRate, 10, 64)
		if err != nil {
			return ProbedAudio{}, fmt.Errorf("invalid bitrate \"%s\": %w", data.Streams[0].BitRate, ErrInvalidFormat)
		}
	}
	if bitrate == 0 {
		// Calculate bitrate manually.
		// Usually the flac format does not give bitrate information in the stream.
		bitrate = uint64(float64(size)/duration) * 8
	}

	p := ProbedAudio{
		CodecName: data.Streams[0].CodecName,
		BitRate:   bitrate,
		Duration:  time.Duration(duration) * time.Second,
		Size:      size,
		Filename:  filename,
		Tags:      data.Format.Tags,
	}
	p.Group = groupAudio(p)
	return p, nil
}

func groupAudio(audio ProbedAudio) string {
	if audio.CodecName == TypeFlac {
		return GroupFLAC
	}
	if audio.CodecName == TypeMp3 {
		if audio.BitRate >= 320000 {
			return Group320Mp3
		}
		if audio.BitRate >= 128000 {
			return Group128Mp3
		}
	}
	if audio.CodecName == TypeAAC {
		if audio.BitRate >= 320000 {
			return Group320aac
		}
		if audio.BitRate >= 256000 {
			return Group256aac
		}
	}
	return LowQuality
}

type stream struct {
	CodecName string `json:"codec_name"`
	BitRate   string `json:"bit_rate"`
}

type format struct {
	Duration string `json:"duration"`
	Size     string `json:"size"`
	Tags     Tags   `json:"tags"`
}

type Tags struct {
	Album     string `json:"album,omitempty"`
	Title     string `json:"title,omitempty"`
	Genre     string `json:"genre,omitempty"`
	Publisher string `json:"publisher,omitempty"`
	Track     string `json:"track,omitempty"`
}

type ProbeResult struct {
	Streams []stream `json:"streams"`
	Format  format   `json:"format"`
}

type ProbedAudio struct {
	CodecName string        `json:"codec_name,omitempty"`
	BitRate   uint64        `json:"bit_rate,omitempty"`
	Duration  time.Duration `json:"duration,omitempty"`
	Size      uint64        `json:"size,omitempty"`
	Filename  string        `json:"filename,omitempty"`
	Tags      Tags          `json:"tags,omitzero"`
	Group     string        `json:"-"`
}

func (r ProbedAudio) Print() {
	slog.Info("Audio",
		slog.String("file", r.Filename),
		slog.String("codec", r.CodecName),
		slog.String("length", r.Duration.String()),
		slog.Uint64("bitrate", r.BitRate/1000),
		slog.String("size", humanize.Bytes(r.Size)))
}

func (r ProbedAudio) Location() string {
	return LocationDir(r.Filename)
}

func LocationDir(file string) string {
	dir := filepath.Dir(file)
	return ungroupDir(dir)
}

func ungroupDir(dir string) string {
	if _, ok := Groups[filepath.Base(dir)]; ok {
		return filepath.Dir(dir)
	}
	return dir
}

func (r ProbedAudio) Basename() string {
	return extractBasename(r.Filename)
}

func extractBasename(path string) string {
	basename := filepath.Base(path)
	return strings.TrimSuffix(basename, filepath.Ext(basename))
}
