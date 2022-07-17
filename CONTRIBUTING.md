## Git Flow for Dev Work
Install go 1.18

Fork the project on github and install golang
```bash
go get github.com/asobti/kube-monkey
git remote rename origin upstream
git remote add origin https://github.com/<YOURUSERNAME>/kube-monkey
git checkout --track -b feature/branchname
```
Then code & stuff. 

Make sure to test your branch from scratch and run `make test`!
