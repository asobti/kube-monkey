## Git Flow for Dev Work
Fork the project on github and install golang
```bash
go get github.com/asobti/kube-monkey
git remote rename origin upstream
git remote add origin https://github.com/<YOURUSERNAME>/kube-monkey
git checkout --track -b feature/branchname
```
Then code & stuff. 

After you think you're done working, make sure to test and get proof of working output (or write tests).

Make sure to test your branch from scratch!

Open a PR and post your output proof

---
## Ways to contribute

- Add unit [tests](https://golang.org/pkg/testing/)
- Support more k8 types
  - ~~deployments~~
  - ~~statefulsets~~
  - dameonsets
  - etc
