# snapshot-cli

Command-line client for OpenStack shared filesystems (Manila) and block storage (Cinder).

## Usage

```
snapshot-cli [command]
```

### Global Flags

| Flag | Description | Default |
| --- | --- | --- |
| `--debug` | enable debug mode | |
| `--output` | output format: json, table | `json` |
| `-h`, `--help` | help for snapshot-cli | |
| `-v`, `--version` | version for snapshot-cli | |

---

## Available Commands

### `cleanup`

Cleanup snapshots.

**Usage:**
```
snapshot-cli cleanup [flags]
```

**Flags:**

| Flag | Description | Default |
| --- | --- | --- |
| `--share` | list shared filesystem snapshots | |
| `--volume` | list volume snapshots | |
| `--volume-id` | ID of the volume to snapshot | |
| `--share-id` | ID of the shared filesystem to snapshot | |
| `--older-than` | Duration to identify old snapshots, e.g. 168h (7 days), 720h (30 days) | `168h0m0s` |
| `-h`, `--help` | help for cleanup | |

---

### `nfs`

Manage shared filesystems storage.

**Usage:**
```
snapshot-cli nfs [command]
```

**Subcommands:**

*   `get`: Get nfs storage information
*   `list`: List block storage resources

#### `nfs get`

Get nfs storage information.

**Usage:**
```
snapshot-cli nfs get [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `--share-id` | ID of the block storage volume to retrieve |
| `-h`, `--help` | help for get |

#### `nfs list`

List block storage resources.

**Usage:**
```
snapshot-cli nfs list [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `-h`, `--help` | help for list |

---

### `snapshot`

Snapshot management commands.

**Usage:**
```
snapshot-cli snapshot [command]
```

**Subcommands:**

*   `create`: Create a snapshot of a volume or shared filesystem
*   `delete`: Delete a snapshot
*   `get`: Get details of a snapshot
*   `list`: List snapshots

#### `snapshot create`

Create a snapshot of a volume or shared filesystem.

**Usage:**
```
snapshot-cli snapshot create [flags]
```

**Flags:**

| Flag | Description | Default |
| --- | --- | --- |
| `--volume-id` | ID of the volume to snapshot | |
| `--share-id` | ID of the shared filesystem to snapshot | |
| `--force` | Force snapshot creation (block only) | |
| `--cleanup` | Cleanup old snapshots after creation | |
| `--older-than` | Duration to identify old snapshots, e.g. 168h (7 days), 720h (30 days) | `168h0m0s` |
| `--name` | Name of the snapshot | |
| `--description` | Description of the snapshot | |
| `--output` | Output format: json, table | `json` |
| `-h`, `--help` | help for create | |

#### `snapshot delete`

Delete a snapshot.

**Usage:**
```
snapshot-cli snapshot delete [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `--share` | list shared filesystem snapshots |
| `--volume` | list volume snapshots |
| `--snapshot-id` | ID of the snapshot to delete |
| `--output` | Output format: json, table |
| `-h`, `--help` | help for delete |

#### `snapshot get`

Get details of a snapshot.

**Usage:**
```
snapshot-cli snapshot get [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `--share-id` | ID of the shared filesystem associated with the snapshot |
| `--volume-id` | ID of the volume associated with the snapshot |
| `--output` | Output format: json, table |
| `-h`, `--help` | help for get |

#### `snapshot list`

List snapshots.

**Usage:**
```
snapshot-cli snapshot list [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `--share` | list shared filesystem snapshots |
| `--volume` | list volume snapshots |
| `--output` | Output format: json, table |
| `-h`, `--help` | help for list |

---

### `volumes`

Manage blockstorage volumes.

**Usage:**
```
snapshot-cli volumes [command]
```

**Subcommands:**

*   `get`: Get block storage information
*   `list`: List block storage resources
*   `snapshot`: Create a snapshot of a block storage volume

#### `volumes get`

Get block storage information.

**Usage:**
```
snapshot-cli volumes get [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `--volume-id` | ID of the block storage volume to retrieve |
| `-h`, `--help` | help for get |

#### `volumes list`

List block storage resources.

**Usage:**
```
snapshot-cli volumes list [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `-h`, `--help` | help for list |

#### `volumes snapshot`

Create a snapshot of a block storage volume.

**Usage:**
```
snapshot-cli volumes snapshot [flags]
```

**Flags:**

| Flag | Description |
| --- | --- |
| `--volume-id` | ID of the block storage volume to snapshot |
| `--snapshot-name` | Name of the snapshot (optional) |
| `--snapshot-dscr` | Description of the snapshot (optional) |
| `--force` | Force snapshot creation |
| `-h`, `--help` | help for snapshot |
