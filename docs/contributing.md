# Contributing

## Setting up

Install Go. The version the module targets is in
[`go.mod`](https://github.com/asobti/kube-monkey/blob/master/go.mod).

Fork the project on GitHub, then:

```bash
git clone https://github.com/asobti/kube-monkey.git
cd kube-monkey
git remote rename origin upstream
git remote add origin https://github.com/<YOURUSERNAME>/kube-monkey
git checkout --track -b feature/branchname
```

Then code and stuff.

## Building and testing

```bash
make build      # build the binary
make container  # build the Docker image
make test       # run the tests
```

Make sure to test your branch from scratch and run `make test` before opening a pull request.

## Working on these docs

The site is built with [MkDocs](https://www.mkdocs.org/) and
[Material for MkDocs](https://squidfunk.github.io/mkdocs-material/). The pages live in
`docs/` and the navigation is in `mkdocs.yml`.

```bash
python3 -m venv .venv
. .venv/bin/activate
pip install -r requirements-docs.txt
mkdocs serve
```

That serves the site at <http://127.0.0.1:8000> and reloads as you edit.

Before you push:

```bash
mkdocs build --strict
```

`--strict` turns broken internal links and unrecognised nav entries into errors. CI runs the
same command.

!!! note "The README is deliberately short"

    Long-form documentation belongs on this site, not in `README.md`. The README's job is to
    tell a visitor what kube-monkey is and send them here.

## Releasing

See [Releasing](releasing.md).
