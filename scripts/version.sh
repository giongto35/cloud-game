#!/bin/sh

file="$1"
version="$2"
from="(<span id=\"v\">).*?(<\/span>)"

# Prints app version derived from git in the following format:
#
# v2-1-gcafeb
# ^  ^ ^ ^
# |  | | |
# |  | | '- commit hash of HEAD (1st 5 symbols)
# |  | '-- "g" stands for git
# |  '---- n commits since last tag
# '------- last tag
#
# See: https://git-scm.com/docs/git-describe
#
# Replaces version placeholder when the file param is provided.
# Uses a provided version value instead of git.
#
if [ "$version" == "" ]; then
  version=$(git describe --abbrev=5 --always --tags 2> /dev/null)
fi
if [ "$file" != "" ]; then
  sed -i -E "s/$from/\1$version\2/" "$file"
  echo "$file $version"
else
  echo "$version"
fi
