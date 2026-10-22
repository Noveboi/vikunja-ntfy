# Vikunja/ntfy.sh Plugin

A simple [Vikunja](https://vikunja.io/) plugin that listens for various event and consequently publishes messages to a [ntfy.sh](https://ntfy.sh/) instance. 

This is a personal project that I use for my self-hosted setup and mainly acts as my introduction to the Go language. Therefore this is not a "production-grade" project and I have no intention of making it so.

## Features

- Publish notifications when one or more tasks are overdue
- Publish notifications for task reminders

## Usage

To use the plugin in your Vikunja instance, you have to put the [`main.go`](src/main.go) file inside your instance's "plugins" directory. Depending on your installation method (bare-metal, Docker, etc...) the intermediate steps might differ. Read the [Vikunja plugin documentation](https://vikunja.io/docs/plugins/#installation-and-configuration) for details. If your Vikunja instance is a Docker container, you can refer to the [sample `compose.yaml` file](compose.yaml) to see how you could set up the plugin.

Use **yaegi** as the loader!

## Important Files

- [`main.go`](src/main.go) contains the entire code of the plugin
- [`compose.yaml`](compose.yaml) is a sample Docker compose file with definitions for a Vikunja and ntfy service. You can use this locally to bootstrap a Vikunja and ntfy instance and test the functionality of the plugin!
