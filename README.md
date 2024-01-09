# maintainers.txt

Parses `maintainers.txt` files in eLife Github repositories, printing a simple report of *what* is maintained by *whom*.

If any repository has no maintainers, the script will exit with a failure.

Accepts an optional input file mapping a `maintainer => alias`. This will replace the name of the maintainer output with 
something else (like an email address).

If an alias map was given and any repository has a maintainer not present in the map, the script will exit with a failure.

## requisites

* Go 1.20+
* A Github Personal Access Token the `repo` scope and access to private repositories.
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
{"jdoe": "john.doe@example.org"}
```

## Licence

Copyright © 2024 eLife Sciences

Distributed under the GNU Affero General Public Licence, version 3.
