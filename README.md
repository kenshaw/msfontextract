# About

Command `msfontextract` extracts Microsoft Windows fonts from a Windows ISO.

It reads the `install.wim` image directly from the ISO, so it does not need
Windows, root access, or a mounted image. It works with Windows 10 and Windows
11 installation ISOs, which can be downloaded from Microsoft's website.

## Installing

```sh
$ go install github.com/kenshaw/msfontextract@latest
```

## Usage

```sh
# extract fonts to ~/.fonts/msfonts and refresh the font cache
$ msfontextract --refresh ~/Downloads/Win11_25H2_English_x64.iso

# show the available editions and what would be extracted, without writing files
$ msfontextract --dry-run ~/Downloads/Win11_25H2_English_x64.iso

# extract fonts from a different edition to a different directory
$ msfontextract --edition '^Windows 11 Education$' --dest ~/fonts ~/Downloads/Win11_25H2_English_x64.iso

$ msfontextract --help
msfontextract a font extraction tool for Microsoft Windows ISOs

Usage:
  msfontextract [flags] <windows iso>

Flags:
      --edition regexp  windows edition (default: ^Windows [0-9]+ Pro$)
      --extract regexp  extract files (default: (?i)^windows/fonts/[^.]+.tt[fc]$)
      --dest string     destination directory (default: ~/.fonts/msfonts)
      --refresh         refresh fonts
  -n, --dry-run         show what would be extracted without extracting
  -v, --version         show version, then exit
  -h, --help            show help, then exit
```

`msfontextract` lists the Windows editions in the ISO, marking the edition
matched by `--edition` with `*`, followed by each extracted file:

```text
editions:
     1: Windows 11 Home
     2: Windows 11 Home N
     3: Windows 11 Home Single Language
     4: Windows 11 Education
     5: Windows 11 Education N
  *  6: Windows 11 Pro
     7: Windows 11 Pro N
     8: Windows 11 Pro Education
     9: Windows 11 Pro Education N
    10: Windows 11 Pro for Workstations
    11: Windows 11 Pro N for Workstations
Windows/Fonts/arial.ttf -> /home/user/.fonts/msfonts/arial.ttf
Windows/Fonts/arialbd.ttf -> /home/user/.fonts/msfonts/arialbd.ttf
...
```

When more than one edition matches `--edition`, the first match is used.

`--refresh` runs `fc-cache` after extracting, so the fonts are available
immediately.
