/*
   Copyright (C) 2023 eLife Sciences

   This program is free software: you can redistribute it and/or modify
   it under the terms of the GNU Affero General Public License as
   published by the Free Software Foundation, either version 3 of the
   License, or (at your option) any later version.

   This program is distributed in the hope that it will be useful,
   but WITHOUT ANY WARRANTY; without even the implied warranty of
   MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
   GNU Affero General Public License for more details.

   You should have received a copy of the GNU Affero General Public License
   along with this program.  If not, see <https://www.gnu.org/licenses/>.
*/

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"

	"github.com/google/go-github/v52/github"
	"golang.org/x/oauth2"
)

type Maintainer = string
type Alias = string
type Project = string

// --- utils

// write templated message `tem` with `args` to stderr
func stderr(tem string, args ...interface{}) {
	fmt.Fprintln(os.Stderr, fmt.Sprintf(tem, args...))
}

// panics with `msg` if `b` is `false`
func ensure(b bool, msg string) {
	if b == false {
		panic(msg)
	}
}

// panics with a "failed with 'blah' while doing 'something'" message
func panicOnErr(err error, action string) {
	if err != nil {
		panic(fmt.Sprintf("failed with '%s' while %s", err.Error(), action))
	}
}

// converts most data to a JSON string with sorted keys.
func as_json(thing interface{}) string {
	json_blob_bytes, err := json.Marshal(thing)
	panicOnErr(err, "marshalling JSON data into a byte array")
	var out bytes.Buffer
	json.Indent(&out, json_blob_bytes, "", "  ")
	return out.String()
}

// returns `true` if file at `path` exists.
func file_exists(path string) bool {
	_, err := os.Stat(path)
	if err != nil && errors.Is(err, os.ErrNotExist) {
		return false
	}
	return true
}

// reads the contents of text file at `path`.
// returns an empty string on any error.
func slurp(path string) string {
	txt, err := ioutil.ReadFile(path)
	if err != nil {
		stderr("failed to read file contents: %s", path)
		return ""
	}
	return string(txt)
}

// read the contents at the given `url`.
// returns an empty string on any error.
func slurp_url(url string, token string) string {
	client := &http.Client{}
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Add("Authorization", fmt.Sprintf("Bearer "+token))
	resp, err := client.Do(req)

	if err != nil {
		stderr("failed to read URL contents: %s", url)
		return ""
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 && resp.StatusCode != 404 {
		stderr("non-200 response from URL: %s (%d)", url, resp.StatusCode)
		return ""
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		stderr("failed to read URL contents: %s", url)
		return ""
	}
	return string(body)
}

// writes `contents` to file at `path`
func spit(contents string, path string) {
	err := os.WriteFile(path, []byte(contents), 0644)
	if err != nil {
		stderr("failed to write file: %s", path)
	}
}

// ---

// pulls a Github personal access token (PAT) out of an envvar `GITHUB_TOKEN`
// panics if token does not exist.
func github_token() string {
	token, present := os.LookupEnv("GITHUB_TOKEN")
	ensure(present, "envvar GITHUB_TOKEN not set.")
	return token
}

// returns `true` if the given `maintainer` looks like a slack channel
// slack channels are supported by eLife CI but I (personally) think it mixes
// purposes: project ownership and CI notifications.
func slack_channel(maintainer string) bool {
	return maintainer != "" && maintainer[0] == '#'
}

// parses the raw contents of a maintainers.txt file,
// replacing the maintainer name with an alias from `maintainer_alias_map`.
// returns a list of maintainers/aliases.
func parse_maintainers_txt_file(contents string, maintainer_alias_map map[Maintainer]Alias) []Alias {
	maintainer_list := []Alias{}
	contents = strings.TrimSpace(contents)
	if contents == "" {
		return maintainer_list
	}
	for _, maintainer := range strings.Split(contents, "\n") {
		if slack_channel(maintainer) {
			// stderr("skipping slack channel: %s", maintainer)
			continue
		}
		alias, present := maintainer_alias_map[maintainer] // jdoe => john.doe@example.org
		if !present {
			alias = maintainer
		}
		maintainer_list = append(maintainer_list, alias)
	}
	return maintainer_list
}

// parses the optional JSON input file of maintainer IDs to an alias.
// input is a simple JSON map: {"jdoe": "john.doe@example.org"}
// returns a map of `maintainer => alias`.
func parse_maintainers_alias_file(path string) map[Maintainer]Alias {
	ensure(file_exists(path), "file does not exist: "+path)

	json_blob := slurp(path)
	ensure(json_blob != "", "file is empty: "+path)

	alias_map := map[Maintainer]Alias{}
	err := json.Unmarshal([]byte(json_blob), &alias_map)
	panicOnErr(err, "deserialising JSON into a map of string=>string")

	return alias_map
}

// fetches all github repositories for `org_name` using the personal access `token`.
func fetch_repos(org_name, token string) []*github.Repository {

	ctx := context.Background()
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)

	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// https://docs.github.com/en/rest/repos/repos?apiVersion=2022-11-28#list-organization-repositories
	opts := &github.RepositoryListByOrgOptions{
		Type: "all",
		Sort: "created",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}
	var repo_list []*github.Repository
	for {
		repo_list_page, resp, err := client.Repositories.ListByOrg(ctx, org_name, opts)
		panicOnErr(err, "listing org repositories")
		repo_list = append(repo_list, repo_list_page...)
		if resp.NextPage == 0 {
			break
		}
		opts.Page = resp.NextPage
	}

	return repo_list
}

func main() {
	args := os.Args[1:]

	token := github_token()
	org_name := "elifesciences"

	// "jdoe" => "john.doe@exaple.org"
	maintainer_alias_map := map[Maintainer]Alias{}
	if len(args) > 0 {
		maintainer_alias_map = parse_maintainers_alias_file(args[0])
	}
	// "jdoe" => "john.doe@example.org" becomes "john.doe@example.org" => "jdoe"
	reverse_maintainer_alias_map := map[Alias]Maintainer{}
	for maintainer, alias := range maintainer_alias_map {
		reverse_maintainer_alias_map[alias] = maintainer
	}

	// step 1, slurp all the maintainers.txt files from all repositories.
	// we cache their contents on disk for development only.

	raw_maintainers := map[Project]string{}
	for _, repo := range fetch_repos(org_name, token) {
		if repo.GetArchived() {
			continue
		}

		// "github-repo-security-alerts--maintainers.txt
		filetem := "%s--maintainers.txt"
		path := fmt.Sprintf(filetem, repo.GetName())

		if file_exists(path) {
			raw_maintainers[repo.GetName()] = slurp(path)
		} else {
			// https://raw.githubusercontent.com/elifesciences/github-repo-security-alerts/master/maintainers.txt
			urltem := "https://raw.githubusercontent.com/%s/%s/%s/maintainers.txt"
			url := fmt.Sprintf(urltem, org_name, repo.GetName(), repo.GetDefaultBranch())

			maintainers_file_contents := slurp_url(url, token)
			spit(maintainers_file_contents, path)
			raw_maintainers[repo.GetName()] = maintainers_file_contents
		}
	}

	// step 2, parse that raw maintainers.txt content into a map of project=>maintainer-list

	// we want a datastructure like: {project1: [maintainer1, maintainer2], project2: [...], ...}
	project_maintainers := map[Project][]Maintainer{}
	for project, maintainers_file_contents := range raw_maintainers {
		project_maintainers[project] = parse_maintainers_txt_file(maintainers_file_contents, maintainer_alias_map)
	}

	// step 3, final checks.
	// 1. projects with no maintainers.txt files should cause script to fail.
	// 2. projects with a maintainer not present in the given alias map (if any)
	//    should cause script to fail.
	fail := false
	for project, maintainer_alias_list := range project_maintainers {
		if len(maintainer_alias_list) == 0 {
			stderr("project has no maintainers: %s", project)
			fail = true
		}
		if len(maintainer_alias_map) > 0 {
			for _, maintainer_alias := range maintainer_alias_list {
				_, present := reverse_maintainer_alias_map[maintainer_alias]
				if !present {
					// "project 'foo' has an unknown maintainer: john"
					stderr("project '%s' has an unknown maintainer: %s", project, maintainer_alias)
					fail = true
				}
			}
		}
	}

	fmt.Println(as_json(project_maintainers))

	if fail {
		os.Exit(1)
	}
}
