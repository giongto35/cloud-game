#!/bin/sh

file="$1"
version="$2"
from="(<span id=\"v\">).*?(<\/span>)"

# Prints app version derived from git in the following format:
#
# v2-1-gcafejk
# ^  ^ ^ ^
# |  | | |
# |  | | '- commit hash of HEAD (1st 7 symbols)
# |  | '-- "g" stands for git
# |  '---- n commits since the last tag
# '------- last tag
#
# See: https://git-scm.com/docs/git-describe
#
# The first input param replaces the version placeholder in the provided file.
# The second input param forces the use of some version value instead of a git-derived one.
#
if [ "$version" = "" ]; then
  version=$(git describe --abbrev=7 --always --tags 2> /dev/null)
fi
if [ "$file" != "" ]; then
  sed -i -E "s/$from/\1$version\2/" "$file"
  echo "$file $version"
else
  echo "$version"
fi
