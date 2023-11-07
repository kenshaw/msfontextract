# About

Command `msfontextract` extracts Microsoft Windows fonts from a ISO.

## Usage

```sh
$ msfontextract ~/Downloads/Win11_23H2_English_x64.iso --dest ~/.fonts/msfonts

$ msfontextract --help
msfontextract, the Microsoft Windows ISO font extraction tool

Usage:
  msfontextract  [flags]

Flags:
      --dest string      destination directory (default "~/.fonts/msfonts")
      --edition string   windows edition (default "^Windows [0-9]+ Pro$")
  -h, --help             help for msfontextract
      --refresh          refresh (default true)
  -v, --version          version for msfontextract
```
