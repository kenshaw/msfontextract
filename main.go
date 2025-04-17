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
	"github.com/mogaika/udf"
	"github.com/xo/ox"
)

var (
	name    = "msfontextract"
	version = "0.0.0-dev"
)

func main() {
	ox.DefaultVersionString = version
	var args Args
	ox.RunContext(
		context.Background(),
		ox.Usage(name, "a font extraction tool for Microsoft Windows ISOs"),
		ox.From(&args),
		ox.Defaults(),
		ox.Spec("[flags] <windows iso>"),
		ox.ValidArgs(1, 1),
		ox.Exec(func(ctx context.Context, cliargs []string) error {
			u, err := user.Current()
			if err != nil {
				return err
			}
			dest := expand(u.HomeDir, args.Dest)
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
			if err := extract(dest, cliargs[0], args.Edition, args.Extract); err != nil {
				return err
			}
			if args.Refresh {
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
}

// extract extracts the matching ttfs to the out directory for the specified
// windows edition regexp.
func extract(out, name string, edition, extract *regexp.Regexp) error {
	f, err := os.OpenFile(name, os.O_RDONLY, 0)
	if err != nil {
		return err
	}
	u := udf.NewUdfFromReader(f)
	var sourcesFE *udf.FileEntry
	for _, f := range u.ReadDir(nil) {
		if strings.ToLower(f.Name()) == "sources" {
			sourcesFE = f.FileEntry()
			break
		}
	}
	if sourcesFE == nil {
		return errors.New("unable to find sources directory")
	}
	var installWimReader io.ReaderAt
	for _, f := range u.ReadDir(sourcesFE) {
		if strings.ToLower(f.Name()) == "install.wim" {
			installWimReader = f.NewReader()
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
	for _, i := range r.Image {
		if edition.MatchString(i.Name) {
			image = i
			break
		}
	}
	if image == nil {
		return fmt.Errorf("unable to find windows edition %q", edition)
	}
	root, err := image.Open()
	if err != nil {
		return err
	}
	return walk(out, "", root, extract)
}

// walk walks the directory recursively, extracting files matching the regexp.
func walk(out string, name string, f *wim.File, re *regexp.Regexp) error {
	if f.IsDir() {
		files, err := f.Readdir()
		if err != nil {
			return err
		}
		for _, fi := range files {
			if err := walk(out, path.Join(name, fi.Name), fi, re); err != nil {
				return err
			}
		}
		return nil
	}
	if re.MatchString(name) {
		r, err := f.Open()
		if err != nil {
			return err
		}
		w, err := os.OpenFile(filepath.Join(out, f.Name), os.O_CREATE|os.O_TRUNC|os.O_WRONLY|os.O_APPEND, 0o644)
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
