#!/bin/bash

# tag_submodules takes in one argument - the bumped version to tag. It will iterate through all submodules in the repo
# and tag each submodule with the bumped version, allowing others to import these submodules more easily.
# This function does not tag the root module though, leaving it to the other commands in make tag-release to handle it
function tag_submodules() {
  echo "Tagging all submodules with version $1..."
  # Look for all subdirectories with `go.mod` - we'll exclude the root module and do this manually
  SUBMODULE_DIRS=$(find * -type f -name "go.mod" | xargs dirname | grep -v "vendor" | grep -v "mk-include" | grep -Fxv "." | sort | uniq)

  # Tag each submodule with its directory name (assume that the package declaration in go.mod = its directory path)
  for submodule_dir in $SUBMODULE_DIRS; do
    if [[ ! $(git tag -l "$submodule_dir/$1") ]]; then
      echo "Tagging $submodule_dir to version $1..."
      git tag $submodule_dir/$1
    else
      echo "Tag $submodule_dir/$1 already exists. Skipping..."
    fi
  done

  # Push tags to remote
  git push --tags
  echo "Submodule tagging complete, will tag the root module in another script."
}
export -f tag_submodules

# Tag all submodules - we'll let the main CI take care of tagging the root module
tag_submodules $1
