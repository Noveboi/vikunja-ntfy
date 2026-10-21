# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/2.0.0/), and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [1.2.0] - 2026-10-21

### Added
- Configuration to allow user to customize the title of the notification (`plugins.ntfy.overdue_title`)

### Changed
- Configuration variable `plugins.ntfy.logUrlAndTopic` to `plugins.ntfy.log_sensitive`

### Tests
- Extensive unit tests for `buildNtfyUrl` function
- Extensive unit tests for `buildNtfyRequest` function

### Refactoring
- Many new pure functions have been added to separate responsibilities and increase testability

## [1.1.0] - 2026-10-20

### Added
- Unit tests for some components in adjacent [`main_test.go`](src/main_test.go)

### Changed
- Notification message content. For 1 overdue task, the message now contains that task's title in full. For more than 1 overdue tasks, the message stays "You have X overdue tasks"
- [`compose.yaml`](compose.yaml) bind mount for the plugin directory to include only the `main.go` file (and not include the `main_test.go` file) 

## [1.0.0] - 2026-10-20

1.0.0 is the initial release for this plugin (🥳🥳). A barebones implementation that works. The plugin at this version only listens for the [TasksOverdueEvent](https://github.com/go-vikunja/vikunja/blob/2eb21632d3fd4cc54f80a51e0e20f0a9a45c0035/pkg/models/events.go#L215) and performs actions to ultimately publish a message to a ntfy.sh topic. Minimal configuration exists for the user to customize the ntfy URL, Topic and logging visibility.

### Added
- [`src/main.go`](src/main.go): the plugin's entire codebase.
- A sample [`compose.yaml`](compose.yaml) file containing a `Vikunja` and `ntfy` service. To be used in testing the plugin locally
- A [`permissions.sh`](permissions.sh) file that sets the appropriate permissions recursively for the `_vikunja_data` folder
- minimal [`.gitignore`](.gitignore) that does not version the `_vikunja_data` folder used by the sample `compose.yaml` file.
