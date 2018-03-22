# generate package from current git master

Just execute `./generate_packages.sh`.
This will generate packages derived from the current git master.
Here the epoch will be set to 0, this means the git master versions are always treated lower than the released versions by dpkg/apt!

# generate release package

* make sure debian/changelog already includes the correct version and changelog entries BEFORE tagging!
    * e.g. `dch -v 1.2-1`
* tag version e.g. `git tag v1.2-1`
* execute `./generate_packages.sh v1.2-1`
* sign
* upload to repository
* dont forget to push the tag with `git push --tags`
