# Hexxy

A modern alternative to `xxd` and `hexdump`

Huge thanks to ![igoracmelo](https://github.com/igoracmelo/xx) for the idea and reference.
The idea and code for colorizing xxd/hexdump output in a gradient format came from them.

![](img.png)

## Example usage
```sh
hexxy /path/to/file.bin
# dont output with color
hexxy --no-color /path/to/file.bin
# refer to multiple files
hexxy file1 file2 file3
# read from stdin
cat mybinary | hexxy
# display offset in Decimal format
hexxy -td file.bin
```
