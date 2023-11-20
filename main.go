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
	"github.com/spf13/cobra"
)

var (
	name    = "msfontextract"
	version = "0.0.0-dev"
)

func main() {
	if err := run(context.Background(), name, version, os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "error: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, appName, appVersion string, cliargs []string) error {
	u, err := user.Current()
	if err != nil {
		return err
	}
	var dest string
	var refresh bool
	var edition string
	var restr string
	c := &cobra.Command{
		Use:     appName + " [flags] <windows iso>",
		Short:   appName + ", the Microsoft Windows ISO font extraction tool",
		Version: appVersion,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			re, err := regexp.Compile(restr)
			if err != nil {
				return fmt.Errorf("unable to compile %q: %v", restr, err)
			}
			dest := expand(u.HomeDir, dest)
			if err := os.MkdirAll(dest, 0o755); err != nil {
				return err
			}
			if err := extract(dest, args[0], edition, re); err != nil {
				return err
			}
			if refresh {
				return exec.Command("fc-cache").Run()
			}
			return nil
		},
	}
	c.Flags().StringVar(&dest, "dest", "~/.fonts/msfonts", "destination directory")
	c.Flags().BoolVar(&refresh, "refresh", true, "refresh")
	c.Flags().StringVar(&edition, "edition", "^Windows [0-9]+ Pro$", "windows edition")
	c.Flags().StringVar(&restr, "regexp", `(?i)^windows/fonts/[^\.]+\.tt[fc]$`, "extract regexp")
	c.SetVersionTemplate("{{ .Name }} {{ .Version }}\n")
	c.InitDefaultHelpCmd()
	c.SetArgs(cliargs[1:])
	c.SilenceErrors, c.SilenceUsage = true, false
	return c.ExecuteContext(ctx)
}

// extract extracts the matching ttfs to the out directory for the specified
// windows edition regexp.
func extract(out, name, edition string, re *regexp.Regexp) error {
	editionRE, err := regexp.Compile(edition)
	if err != nil {
		return err
	}
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
		if editionRE.MatchString(i.Name) {
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
	return walk(out, "", root, re)
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
