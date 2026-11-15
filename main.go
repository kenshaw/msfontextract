// Command msfontextract extracts Microsoft Windows fonts from a ISO.
package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"os/user"
	"path"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/Microsoft/go-winio/wim"
	"github.com/xo/ox"
	"golift.io/udf"
)

var (
	name    = "msfontextract"
	version = "0.0.0-dev"
)

func main() {
	var args Args
	ox.RunContext(
		context.Background(),
		ox.Usage(name, "a font extraction tool for Microsoft Windows ISOs"),
		ox.From(&args),
		ox.Defaults(),
		ox.VersionString(version),
		ox.Spec("[flags] <windows iso>"),
		ox.ValidArgs(1, 1),
		ox.Exec(func(ctx context.Context, cliargs []string) error {
			u, err := user.Current()
			if err != nil {
				return err
			}
			dest := expand(u.HomeDir, args.Dest)
			if !args.DryRun {
				if err := os.MkdirAll(dest, 0o755); err != nil {
					return err
				}
			}
			if err := extract(dest, cliargs[0], args.Edition, args.Extract, args.DryRun); err != nil {
				return err
			}
			if args.Refresh && !args.DryRun {
				return exec.Command("fc-cache").Run()
			}
			return nil
		}),
	)
}

type Args struct {
	Edition *regexp.Regexp `ox:"windows edition,default:^Windows [0-9]+ Pro$"`
	Extract *regexp.Regexp `ox:"extract files,default:(?i)^windows/fonts/[^\\\\.]+\\\\.tt[fc]$"`
	Dest    string         `ox:"destination directory,default:~/.fonts/msfonts"`
	Refresh bool           `ox:"refresh fonts"`
	DryRun  bool           `ox:"show what would be extracted without extracting,short:n"`
}

// extract extracts the matching ttfs to the out directory for the specified
// windows edition regexp, printing the available editions and each extracted
// file. When dryRun is true, no files are written.
func extract(out, name string, edition, extract *regexp.Regexp, dryRun bool) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	u, err := udf.NewUdfFromReader(f)
	if err != nil {
		return err
	}
	rootFiles, err := u.ReadDir(nil)
	if err != nil {
		return err
	}
	var sourcesFE *udf.FileEntry
	for _, f := range rootFiles {
		if strings.ToLower(f.Name()) == "sources" {
			if sourcesFE, err = f.FileEntry(); err != nil {
				return err
			}
			break
		}
	}
	if sourcesFE == nil {
		return errors.New("unable to find sources directory")
	}
	sourcesFiles, err := u.ReadDir(sourcesFE)
	if err != nil {
		return err
	}
	var installWimReader io.ReaderAt
	for _, f := range sourcesFiles {
		if strings.ToLower(f.Name()) == "install.wim" {
			if installWimReader, err = f.NewReader(); err != nil {
				return err
			}
			break
		}
	}
	if installWimReader == nil {
		return errors.New("unable to find install.wim")
	}
	r, err := wim.NewReader(installWimReader)
	if err != nil {
		return err
	}
	defer r.Close()
	var image *wim.Image
	fmt.Println("editions:")
	for _, i := range r.Image {
		marker := " "
		if image == nil && edition.MatchString(i.Name) {
			image, marker = i, "*"
		}
		fmt.Printf("  %s %2d: %s\n", marker, i.Index, i.Name)
	}
	if image == nil {
		return fmt.Errorf("unable to find windows edition %q", edition)
	}
	root, err := image.Open()
	if err != nil {
		return err
	}
	return walk(out, "", root, extract, dryRun)
}

// walk walks the directory recursively, extracting files matching the regexp.
func walk(out string, name string, f *wim.File, re *regexp.Regexp, dryRun bool) error {
	if f.IsDir() {
		files, err := f.Readdir()
		if err != nil {
			return err
		}
		for _, fi := range files {
			if err := walk(out, path.Join(name, fi.Name), fi, re, dryRun); err != nil {
				return err
			}
		}
		return nil
	}
	if re.MatchString(name) {
		dest := filepath.Join(out, f.Name)
		fmt.Printf("%s -> %s\n", name, dest)
		if dryRun {
			return nil
		}
		r, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(dest, os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0o644)
		if err != nil {
			return err
		}
		if _, err := io.Copy(w, r); err != nil {
			_ = w.Close()
			_ = r.Close()
			return err
		}
		if err := w.Close(); err != nil {
			_ = r.Close()
			return err
		}
		if err := r.Close(); err != nil {
			return err
		}
	}
	return nil
}

// expand expands the beginning tilde (~) in a file name to the provided home
// directory.
func expand(homeDir string, name string) string {
	switch {
	case name == "~":
		return homeDir
	case strings.HasPrefix(name, "~/"):
		return filepath.Join(homeDir, strings.TrimPrefix(name, "~/"))
	}
	return name
}
