## Git Flow for Dev Work

Install Go. The version the module targets is in [go.mod](go.mod).

Fork the project on github, then:

```bash
git clone https://github.com/asobti/kube-monkey.git
cd kube-monkey
git remote rename origin upstream
git remote add origin https://github.com/<YOURUSERNAME>/kube-monkey
git checkout --track -b feature/branchname
```

Then code & stuff.

Make sure to test your branch from scratch and run `make test`!

## Docs

The documentation site lives in `docs/`. To work on it:

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install -r requirements-docs.txt
mkdocs serve
```

Long-form documentation belongs on the site, not in `README.md`.
See [Contributing](https://asobti.github.io/kube-monkey/contributing/) for more.

## Releasing

See [RELEASING.md](RELEASING.md).
