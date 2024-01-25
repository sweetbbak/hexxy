# Hexxy

A modern alternative to `xxd` and `hexdump`

Huge thanks to ![igoracmelo](https://github.com/igoracmelo/xx) for to use color gradients
to make bytes more readable.

![example of hexxy in action](img.png)

## Example usage
```sh
hexxy /path/to/file.bin
# dont output with color
hexxy --no-color /path/to/file.bin
# dump multiple files
hexxy file1 file2 file3
# read from stdin
cat mybinary | hexxy
# display offset in Decimal format
hexxy -td file.bin
# display offset in Octal format
hexxy -to file.bin
```
