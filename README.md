# maintainers.txt

Script that parses `maintainers.txt` files in elife repositories and prints a simple report of *what* is maintained by *whom*.

Accepts an optional input file mapping a `maintainer=>alias`. This will replace the name of the maintainer output with 
something else (like an email address).

## requisites

* Go 1.20+
* A Github Personal Access Token the `repo` scope. 
    - See: https://docs.github.com/en/rest/dependabot/alerts#list-dependabot-alerts-for-an-organization

## Installation

    git clone https://github.com/elifesciences/maintainers-txt
    cd maintainers-txt
    go build .

## Usage

    GITHUB_TOKEN=your-github-token ./maintainers-txt

or

    GITHUB_TOKEN=your-github-token ./maintainers-txt alias-map.json

and `alias-map.json` might look like:

```json
{"lsh-0": "l.skibinski@example.org"}
```

## Licence

Copyright © 2023 eLife Sciences

Distributed under the GNU Affero General Public Licence, version 3.
