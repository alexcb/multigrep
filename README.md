# multigrep

Recursively greps all files in current directory for a list of patterns.
If all patterns match, the matched file path is displayed.

## Building

First download earthly.

Then run:

    earthly +all

builds are written to `build/ubuntu/multigrep` and `build/alpine/multigrep`

## Example use

    build/alpine/multigrep grep alexcb

which will display two files which both contained the matched patterns:

    build/alpine/multigrep
    go.mod

## Futurework

- support a `--context=<int>` flag which will limit matches to context sizes
  - display matches in colour to stdout.
- support a per-pattern `-v` option to negate the match
