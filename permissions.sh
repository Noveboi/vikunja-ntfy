# This script sets the appropriate permissions for the Vikunja bind mounts
# Use this if you get a "permission denied" error when running Vikunja locally.
chown -R 1000 ./_vikunja_data
