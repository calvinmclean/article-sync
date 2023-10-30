# Article Sync

Manage your articles using git. Synchronize markdown files from a git repository to [dev.to](https://dev.to).

## Directory Structure
This relies on having a directory structure where each article/post consists of a directory with `article.md` and `article.json`:
```
articles/
└── test-article
    ├── article.json
    └── article.md
```

- `article.md`: this is the markdown contents of the post
- `article.json`: this contains some extra details about the post like the title and ID:
    ```json
    {
        "title": "My New Article",
        "description": "this article is a test"
    }
    ```

Once an article is posted, the ID and slug are saved to the `article.json` file:
```json
{
    "id": 1234,
    "slug": "my-new-article-1234",
    "title": "My New Article",
    "description": "this article is a test"
}
```

## GitHub Action Usage

When opening a PR, comment a summary of changes
```yaml
name: Synchronization summary
on:
  pull_request:
    branches:
      - main
jobs:
  comment:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: calvinmclean/article-sync@v1.0.1
        with:
          type: summary
          api_key: ${{ secrets.DEV_TO_API_KEY }}
```

After pushing to main, synchronize with dev.to and make a commit with new IDs if articles are created
```yaml
name: Synchronize and commit
on:
  push:
    branches:
      - main
jobs:
  commit_file:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      - uses: calvinmclean/article-sync@v1.0.1
        with:
          type: synchronize
          api_key: ${{ secrets.DEV_TO_API_KEY }}
```

This works declaratively by parsing each article and:
- If it does not have an ID, create a new article and save ID
- If it does have an ID:
    - Compare to existing contents fetched by ID
    - Update if changed, otherwise leave alone

## Roadmap
- Support tags
- Allow naming files other than `article.md` or `article.json`
