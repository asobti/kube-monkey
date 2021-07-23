# Contributing new Helm Charts

Note: Unfortunately the process to publish new Helm Charts is not automated yet, so please bear with us while we explain this two step process 🦄

To add a new version of a chart you need 2 pull requests.

## Publish Sources

* Start by forking this Github repository and creating a new branch from master.
* Update the chart under the directory `/helm/kubemonkey`
* Submit a Pull Request with these changes.

## Publish Helm Charts

* Switch to the `gh-pages` branch and create a new branch from this in your fork.
* Package the new version of the chart into a `.tgz` file. See [Helm Package](https://helm.sh/docs/helm/helm_package/).
* Add your file under the `repo` folder.
* Finally, you will need to update the index file. See [Helm Repo Index](https://helm.sh/docs/helm/helm_repo_index/). 
* Submit a Pull Request with these changes.
