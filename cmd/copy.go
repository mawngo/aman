package cmd

import (
	"aman/internal/audio"
	"aman/internal/fileutils"
	"errors"
	"github.com/mawngo/go-maplock"
	"github.com/samber/lo"
	"github.com/spf13/cobra"
	"io/fs"
	"log/slog"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"sync/atomic"
	"time"
)

func newCopyCommand() *cobra.Command {
	f := copyFlags{
		order:       []string{"flac", "320"},
		depth:       -1,
		concurrency: runtime.NumCPU(),
	}

	command := cobra.Command{
		Use:   "copy <src> <target>",
		Short: "Selectively copy music files by bitrate",
		Args:  cobra.ExactArgs(2),
		Run: func(_ *cobra.Command, args []string) {
			if !slices.Contains(f.order, "128") {
				// Always fallback to 128.
				f.order = append(f.order, "128")
			}

			target := args[1]
			if s, err := os.Stat(target); err == nil && !s.IsDir() {
				slog.Error("Target is not a directory", slog.String("target", target))
				return
			}

			slog.Info("Moving files", slog.String("order", strings.Join(f.order, ">")))
			start := time.Now()
			copied := atomic.Int64{}
			lock := maplock.New[string]()
			count, err := audio.ProcessAudio(args[0], func(a audio.ProbedAudio) {
				filename := filepath.Base(a.Filename)
				if f.flat {
					dest := filepath.Join(target, filename)
					if cpy(a, dest, f.order, lock) {
						copied.Add(1)
					}
					return
				}

				parent := filepath.Dir(a.Filename)
				if _, ok := audio.Groups[filepath.Base(parent)]; ok {
					parent = filepath.Dir(parent)
				}

				rel := lo.Must(filepath.Rel(args[0], parent))
				dest := filepath.Join(target, rel, filename)
				if cpy(a, dest, f.order, lock) {
					copied.Add(1)
				}
			})
			if err != nil {
				slog.Error("Error copy audio files", slog.Any("err", err))
				return
			}
			slog.Info("Audio files sorted",
				slog.Int64("count", count),
				slog.Int64("moved", copied.Load()),
				slog.String("took", time.Since(start).String()))
		},
	}
	command.Flags().StringSliceVarP(&f.order, "order", "o", f.order, "Preferred quality order")
	command.Flags().IntVar(&f.depth, "depth", f.depth, "Maximum depth to search for audio files")
	command.Flags().BoolVar(&f.flat, "flat", f.flat, "Flatten directory structure")
	command.Flags().IntVar(&f.concurrency, "concurrency", f.concurrency, "Number of thread to use")
	return &command
}

type copyFlags struct {
	order       []string
	flat        bool
	depth       int
	concurrency int
}

func cpy(a audio.ProbedAudio, dest string, orders []string, lock *maplock.MapLock[string]) bool {
	base := strings.TrimSuffix(dest, filepath.Ext(dest))
	lock.Lock(base)
	defer lock.Unlock(base)
	for _, extension := range audio.SupportedExtensions {
		dest := base + extension
		if _, err := os.Stat(dest); errors.Is(err, fs.ErrNotExist) {
			continue
		}
		destProbe, err := audio.Probe(dest)
		if err != nil {
			slog.Error("Error probing file", slog.String("file", dest), slog.Any("err", err))
			return false
		}
		if slices.Index(orders, destProbe.Group) > slices.Index(orders, a.Group) {
			return fileutils.CopyFile(a.Filename, dest)
		}
		return false
	}
	return fileutils.CopyFile(a.Filename, dest)
}
