# Article Sync

This program allows for easily synchronizing markdown files to a blog/article platform such as [dev.to](https://dev.to).
Currently only [dev.to](https://dev.to) is supported.

It works declaratively by parsing each article and:
- If it does not have an ID, create a new article and save ID
- If it does have an ID:
    - Compare to existing contents fetched by ID
    - Update if changed, otherwise leave alone

I plan to create a Github Action from this program that will allow you to have an articles repository that:
- Does a dry-run when a PR is opened to `main` and comments on the PR so you can see what would change
- When changes are pushed to `main`, synchronize with the platform API and commit updated `article.json` files 

## How To

```shell
go run main.go --api-key "SECRET_API_KEY"
```

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
