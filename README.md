<p align="center"><img src="zackup.png" alt="zackup" style="max-width:100%;"></p>

# zackup - Backup to ZFS


Small utility to replace BackupPC.

- for each `host` in `list_of_hosts` do:
  - if `zfs_dataset_for_host` does not exist:
    - create `zfs_dataset_for_host`
  - for each `command` in `list_of_pre_commands_for_host`:
    - execute `command` on host
  - rsync `list_of_configured_files` from `host` to `zfs_dataset_for_host`
  - for each `command` in `list_of_post_commands_for_host`:
    - execute `command` on `host`
  - create snapshot of `zfs_dataset_for_host`


## Usage

    zackup [COMMAND] [--root ROOT_DIR] [--gelf host:port]

Instead of `--root`, you may set a `ZACKUP_ROOT` environment variable.
The command line flag takes precedence if both are given.

    export ZACKUP_ROOT=ROOT_DIR
    zackup [COMMAND]

Use something like `--gelf graylog.example.com:12201` to direct log
messages to a remote logging server.

### `COMMAND`

Defaults to `help`.

- `run`

  Creates a backup for each host config. Backups are stored locally in
  a per-host dataset.

  Run `zackup help run` for a list of possible options.

- `status`

  Prints a list of hosts and their backup status (last success, size)

- `help`

  Prints a help listing with all available commands.

You can run `zackup help COMMAND` to get a list of possible options.

### `ROOT_DIR`

Defaults to `/usr/local/etc/zackup`. Its content is used as config tree.
We assume these files exist in `ROOT_DIR`:

    ROOT_DIR/
    +-- config.yml                    service configuration
    +-- globals.yml                   global defaults for host configs
    +-- hosts/
        +-- $host/config.yml          host config (variant A)
        +-- $host.yml                 host config (variant B)
        +-- $host/{pre,post}.*.sh     host-specific scripts (optional)

The *list of hosts* is comprised of each `ROOT_DIR/hosts/$host` entry.
A `$host` is a string matching the rules for DNS host name labels.

A host may have scripts executed (via SSH) *before* and/or *after*
rsyncing. These scripts are defined by the `ROOT_DIR/hosts/$host/pre.*.sh`
and `ROOT_DIR/hosts/$host/post.*.sh` files. See sec. "Hooks" below.

You may also create a `ROOT_DIR/hosts/example.com.yml` instead of a
`ROOT_DIR/hosts/example.com/config.yml` if you don't have any pre- or
post-commands to execute (i.e. you have no script *files*, you can still
define *inline* script one-liners).

It is an error to have both `ROOT_DIR/hosts/$host/config.yml` and
`ROOT_DIR/$host.yml`.


## Setup

It is recommended to create a compressed ZFS dataset for all backups:

```console
# zfs create -o compression=lz4 zpool/zackup
```

and add `zpool/zackup` as `root_dataset` to the service configuration file.


## Service config

zackup requires a previously mentioned service configuration file in
`ROOT_DIR/config.yml`, which defines these properties:

```yaml
parallel:     uint8     # number backups to run in parallel
root_dataset: string    # base dataset to create host-datasets under
mount_base:   path      # working directory to mount host dataset into
log_level:    enum      # one of DEBUG, INFO, WARN, ERROR, FATAL, PANIC (case insensitive)
graylog:      addr      # if set, write logs to this GELF UDP endpoint

# We require rsync and ssh commands to be in $PATH. Adjust these, if either
# $PATH does not contain these, or your binaries are named differently.
rsync_bin:    string
ssh_bin:      string

# daemon hold settings specific to the "zackup serve" command.
daemon:
  # strftime format for scheduled backups
  schedule:   "%H:%M:%S"

  # Random amount of jitter around the schedule time. The actual backup
  # will start in the time range schedule ± jitter/2. jitter must be
  # parsable with Go's time.ParseDuration() function.
  #
  # Jitter is applied to each host seperately.
  jitter:     duration
```

The defaults are:

```yaml
parallel:     5
root_dataset: zpool/zackup
mount_base:   /zpool/zackup
log_level:    info
rsync_bin:    rsync
ssh_bin:      ssh
daemon:
  schedule:   "04:00:00"
  jitter:     "40m" # → between 03:40:00 and 04:20:00
```


## Hooks (pre- and post-scripts)

Within the host config directory, you can define `pre.*.sh` and `post.*.sh`
files, which are executed in alphabetically order *before* the rsync
process starts, and *after* it has finished.

For conveniance, you can add *inline* hook scripts into the host config
file (more on that in the next section). These inline scripts are executed
before any `pre.*.sh` or `post.*.sh` scripts.

Please note that hook scripts (both inline and files) are piped directly
into `/bin/sh -esx`, so you don't need a shebang. Think of a simple
`cat $host/pre.*.sh | ssh $host /bin/sh -esx`.

If any of those scripts exits with a non-zero exit status, the backup is
marked as failed.


## Host config

A host's config file is written in YAML and has this structure:

```yaml
ssh:
  user:     string    # username on the remote host
  port:     uint16    # SSH port number
  timeout:  int       # timeout for establishing connection

rsync:
  include:  []string  # rsync pattern for included files/directories
  exclude:  []string  # rsync pattern for excluded files/directories
  args:     []string  # other rsync arguments

  # included, excluded and args will be merged with the global config
  # by default. if you want a fresh start, without globally defined
  # values, uncomment the corresponding entry:
  #override_global_included: true
  #override_global_excluded: true
  #override_global_args:     true

# Inline scripts executed on the remote host before and after rsyncing,
# and before any `pre.*.sh` and/or `post.*.sh` scripts for this host.
pre_script:  string
post_script: string
```


## Global config

zackup looks for a global config file in `ROOT_DIR/globals.yml`.

Use this file to specify defaults (a single host's config is basically
merged into the global config). zackup brings no defaults (not even for
rsync!), but you can use this as a start:

```yaml
ssh:
  user: root
  port: 22
  timeout: 10
rsync:
  include:
  - /etc
  - /home
  - /opt
  - /root
  - /srv
  - /usr/local/etc
  - /var/spool/cron
  - /var/www
  exclude:
  - "tmp"
  - "*.log"
  - "*.log.*"
  - ".cache"
  - ".config"
  args:
  - "--numeric-ids"
  - "--perms"
  - "--owner"
  - "--group"
  - "--devices"
  - "--specials"
  - "--links"
  - "--hard-links"
  - "--block-size=2048"
  - "--recursive"
```

# Copyright

Copyright (C) 2018-2019 Dominik Menke, Digineo GmbH. All rights reserved.

Contains ported code of BackupPC, Copyright (C) 2001-2018 Craig Barret.

This program is free software; you can redistribute it and/or modify it
under the terms of the GNU General Public License.

See the LICENSE file.
