package audio

import (
	"encoding/json"
	"errors"
	"github.com/dustin/go-humanize"
	"github.com/samber/lo"
	"log/slog"
	"os/exec"
	"strconv"
	"time"
)

var Groups = map[string]bool{
	"320":     true,
	"128":     true,
	"flac":    true,
	"unknown": true,
	"ll":      false, // Legacy group.
	"l":       false, // Legacy group.
}

var ErrNotAudioFile = errors.New("not an audio file")

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

	duration := lo.Must(strconv.ParseFloat(data.Format.Duration, 64))
	size := lo.Must(strconv.ParseUint(data.Format.Size, 10, 64))
	bitrate, err := strconv.ParseUint(data.Streams[0].BitRate, 10, 64)
	if bitrate == 0 {
		// Calculate bitrate manually.
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
	if audio.CodecName == "flac" {
		return "flac"
	}
	if audio.CodecName == "mp3" {
		if audio.BitRate >= 320000 {
			return "320"
		}
		return "128"
	}
	return "unknown"
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
	Tags      Tags          `json:"tags,omitempty"`
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
