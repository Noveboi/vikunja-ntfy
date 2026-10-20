# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/2.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

# [1.0.0] - 2026-10-20

1.0.0 is the initial release for this plugin (🥳🥳). A barebones implementation that works. The plugin at this version only listens for the [TasksOverdueEvent](https://github.com/go-vikunja/vikunja/blob/2eb21632d3fd4cc54f80a51e0e20f0a9a45c0035/pkg/models/events.go#L215) and performs actions to ultimately publish a message to a ntfy.sh topic. Minimal configuration exists for the user to customize the ntfy URL, Topic and logging visibility.

## Added
- [`src/main.go`](src/main.go): the plugin's entire codebase.
- A sample [`compose.yaml`](compose.yaml) file containing a `Vikunja` and `ntfy` service. To be used in testing the plugin locally
- A [`permissions.sh`](permissions.sh) file that sets the appropriate permissions recursively for the `_vikunja_data` folder
- minimal [`.gitignore`](.gitignore) that does not version the `_vikunja_data` folder used by the sample `compose.yaml` file.
