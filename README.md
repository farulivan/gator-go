# Gator

A CLI RSS feed aggregator built in Go. Gator lets you register users, add RSS feeds, follow or unfollow them, run a background aggregator to fetch new posts, and browse the latest articles — all from your terminal.

## Table of Contents

- [Prerequisites](#prerequisites)
- [Installation](#installation)
- [Configuration](#configuration)
- [Database Setup](#database-setup)
- [Usage](#usage)
  - [User Management](#user-management)
  - [Feed Management](#feed-management)
  - [Aggregation](#aggregation)
  - [Browsing Posts](#browsing-posts)
- [Commands Reference](#commands-reference)

---

## Prerequisites

Make sure the following are installed on your machine before proceeding:

- **[Go](https://go.dev/dl/)** (1.22 or later)
- **[PostgreSQL](https://www.postgresql.org/download/)** (14 or later)
- **[Goose](https://github.com/pressly/goose)** — database migration tool

  ```bash
  go install github.com/pressly/goose/v3/cmd/goose@latest
  ```

---

## Installation

Install the `gator` binary directly via `go install`:

```bash
go install github.com/farulivan/gator-go@latest
```

Or clone the repository and build from source:

```bash
git clone https://github.com/farulivan/gator-go.git
cd gator-go
go install .
```

Verify the installation:

```bash
gator
# usage: gator <command> [args...]
```

---

## Configuration

Gator reads its configuration from `~/.gatorconfig.json` in your home directory. Create this file before running any commands:

```bash
touch ~/.gatorconfig.json
```

Populate it with your PostgreSQL connection string:

```json
{
  "db_url": "postgres://username:password@localhost:5432/gator?sslmode=disable",
  "current_user_name": ""
}
```

| Field               | Description                                                    |
| ------------------- | -------------------------------------------------------------- |
| `db_url`            | PostgreSQL connection string for your Gator database           |
| `current_user_name` | The currently logged-in user. Managed automatically by Gator.  |

> **Note:** Replace `username`, `password`, and the database name with your actual PostgreSQL credentials.

---

## Database Setup

**1. Create a PostgreSQL database:**

```bash
createdb gator
```

**2. Run database migrations** from the project root using Goose:

```bash
goose -dir sql/schema postgres "postgres://username:password@localhost:5432/gator?sslmode=disable" up
```

This will create all required tables: `users`, `feeds`, `feed_follows`, and `posts`.

---

## Usage

### User Management

**Register a new user** (also sets them as the current user):

```bash
gator register alice
```

**Log in as an existing user:**

```bash
gator login alice
```

**List all registered users** (current user is marked with `(current)`):

```bash
gator users
```

**Reset the database** — deletes all users and their associated data:

```bash
gator reset
```

---

### Feed Management

> Commands in this section require a logged-in user.

**Add a new RSS feed** (automatically followed by the adding user):

```bash
gator addfeed "TechCrunch" https://techcrunch.com/feed/
```

**List all available feeds:**

```bash
gator feeds
```

**Follow an existing feed by its URL:**

```bash
gator follow https://techcrunch.com/feed/
```

**Unfollow a feed by its URL:**

```bash
gator unfollow https://techcrunch.com/feed/
```

**List feeds you are currently following:**

```bash
gator following
```

---

### Aggregation

Start the feed aggregator. It will continuously fetch posts from all registered feeds on the given interval and store them in the database.

```bash
gator agg <interval>
```

The `interval` accepts any valid Go duration string:

```bash
gator agg 30s    # fetch every 30 seconds
gator agg 1m     # fetch every 1 minute
gator agg 5m     # fetch every 5 minutes
```

> Run `agg` in a separate terminal session so it keeps running in the background while you use other commands.

---

### Browsing Posts

Browse the latest posts from feeds you follow. Defaults to showing the 2 most recent posts:

```bash
gator browse
```

Pass an optional limit to see more posts:

```bash
gator browse 10
```

---

## Commands Reference

| Command                    | Auth Required | Description                                          |
| -------------------------- | :-----------: | ---------------------------------------------------- |
| `register <name>`          | No            | Create a new user and set them as current            |
| `login <name>`             | No            | Switch the current user                              |
| `users`                    | No            | List all registered users                            |
| `reset`                    | No            | Delete all users and their data                      |
| `feeds`                    | No            | List all feeds in the database                       |
| `agg <interval>`           | No            | Start the background feed aggregator                 |
| `addfeed <name> <url>`     | Yes           | Add a new RSS feed and auto-follow it                |
| `follow <url>`             | Yes           | Follow an existing feed by URL                       |
| `unfollow <url>`           | Yes           | Unfollow a feed by URL                               |
| `following`                | Yes           | List all feeds followed by the current user          |
| `browse [limit]`           | Yes           | Browse posts from followed feeds (default limit: 2)  |
