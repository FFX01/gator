# Gator
Gator is a CLI tool for managing RSS feeds. You can follow feeds, ingest posts, and browse posts.

## Installation
1. Ensure you have go version >=1.23 installed.
2. Ensure you have postgres installed and running.
3. Install [goose](https://github.com/pressly/goose) for running migrations.
4. Run `go install github.com/FFX01/gator`

## Configuration
1. Create a file called `.gatorconfig.json` in your `$HOME` directory.
2. Copy/paste the contents of `.gatorconfig.json.example` into this file.
3. Update the `db_url` key to point at your postgres database.
4. Create a new database in postgres called `gator`.
5. `cd` into `sql/schema` and run `goose postgres <your db url> up`

## Usage
- Register a new user: `gator register <name>`
- Login as an existing user: `gator login <name>`
- Add a new feed: `gator addfeed "<name>" "<url>"`
    - This will automatically follow this feed for the current user.
- Reset the Db: `gator reset`
    - This will clear all data in the databse, but it will not drop the tables.
- View all users: `gator users`
- View all feeds: `gator feeds`
- View all feeds the current user follows: `gator following`
- Unfollow a feed: `gator unfollow "<url>"`
- Fetch posts: `gator agg <duration>`
    - This will run an infinite loop to continuously fetch data from feeds, starting with the least recently fetched.
    - `duration` should be in the format `<n><unit>`, examples: `10s`, `1m`, `3h`
- View Post title, description, and URL: `gator browse <limit>`
    - `limit` is how many posts to view. Can be omitted and use default of 2.
