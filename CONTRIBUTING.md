## Git Flow for Dev Work
Install go 1.15

Fork the project on github and install golang
```bash
go get github.com/asobti/kube-monkey
git remote rename origin upstream
git remote add origin https://github.com/<YOURUSERNAME>/kube-monkey
git checkout --track -b feature/branchname
```
Then code & stuff. 

After you think you're done working, make sure to test and get proof of working output (or write tests).

Make sure to test your branch from scratch and run `make test`! The make process will gofmt your code. Make sure to commit any files that were modified before opening a PR! Otherwise the CI process will reject your PR for code formatting. When you make the PR, please output your proof.

---
## Ways to contribute

- Add unit [tests](https://golang.org/pkg/testing/)
- Design us a cool logo!
- Support more forms of Chaos
  - ~~deployments~~
  - ~~statefulsets~~
  - ~~dameonsets~~
  - Disabling svc
  - Disabling ingress
  - Disabling configmap
  - Cordoning off Node (chaos-gorilla style)
  - Deleting Node (chaos-gorilla style)
  - etc
- ~~Enhance documentation for [Helm](https://github.com/linki/chaoskube#how)~~
- Add [related projects](https://github.com/linki/chaoskube#related-work)
- Convert from glide to dep
- ~~Push image to dockerhub~~
- Analyze api versions and link dependency chart (i.e. k8s v1.8+ deprecates v1beta deployments (madeup)) #70
- Whitelist opt-in feature #5

