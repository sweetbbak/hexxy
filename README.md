<p align="center">
  <img src="assets/hexxy.png" />
  <div align="center">A modern alternative to `xxd` and `hexdump`</div>
</p>

![example of hexxy in action](assets/img.png)

## Quick install

requirements: Go 1.20+ (it may build with earlier versions as well but I have not tested them) and git

```sh
go install github.com/sweetbbak/hexxy@latest
```

On ArchLinux ([hexxy-git](https://aur.archlinux.org/packages/hexxy-git)), e.g.:

```
pikaur -S hexxy-git
paru -S hexxy-git
yay -S hexxy-git
```

## Example usage

```sh
# normal usage
hexxy /path/to/file.bin

# output without color
hexxy --no-color /path/to/file.bin

# read from stdin
cat mybinary | hexxy

# display plain output
hexxy -p file.bin

# Include a binary as a C variable
hexxy -i input-file > output.c

# Use plain non-formatted output
hexxy -p input-file

# crunch empty lines with a '*' and use uppercase HEX
hexxy -a --upper input-file

# Reverse plain non-formatted output (reverse plain)
hexxy -rp input-file

# Show output with a space in between N groups of bytes
hexxy -g1 input-file ... -> outputs: 00000000: 0f 1a ff ff 00 aa

# display offset in Decimal format
hexxy -td file.bin

# display offset in Octal format
hexxy -to file.bin

# configure color
# shows color even when piping to a file or stdout/stderr
hexxy --color=always # or never, auto

# turn off ascii table color (but keep byte coloring)
hexxy -A

# write the default config file
hexxy --create-config

# ignore config file (you can also just delete it)
# it is not required. Command line flags override config flags
hexxy --no-config

# show ascii table bars
# and set the seperator (great time to set a default in the config file)
hexxy --bars --seperator='|'
```

## Building

```sh
git clone https://github.com/sweetbbak/hexxy.git
cd hexxy
go build -o hexxy -ldflags='-s -w' ./src
# or use just by running 'just'
```

## Changelog

- 3/23/25: added a config file and more options

## Performance

`zk` is a 17mb binary

```sh
xxd -i ~/bin/zk &> /dev/null  0.66s user 0.02s system 99% cpu 0.677 total
hexxy -i ~/bin/zk &> /dev/null  0.16s user 0.01s system 98% cpu 0.165 total
```

```sh
# plain XXD
xxd ~/bin/zk &> /dev/null  0.12s user 0.01s system 99% cpu 0.126 total

# hexxy without color
hexxy -N ~/bin/zk &> /dev/null  0.21s user 0.01s system 100% cpu 0.223 total

# hexxy with color
hexxy ~/bin/zk &> /dev/null  0.37s user 0.01s system 99% cpu 0.383 total
```

`hexxy` is obviously going to be slower as it is writing a lot more bytes in the form of
ANSI escape sequences. There is potential to optimize this using some deduplication or Huffman
encoding, but that might also be slower.

## Credits

thanks to [felixge](https://github.com/felixge/go-xxd) for showing how this is done quickly
thanks to [igoracmelo](https://github.com/igoracmelo/xx) for the idea to colorize hexdump output with a gradient

thanks to everyone who has committed to this repo! <3
