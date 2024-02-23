# Let's Build

Source code for Simon's personal blog:

https://letsbuild.cloud

See [style.md](style.md) for an explanation of my general writing style.

## Installation and pre-requisites

Install Astro and all dependencies.

```sh
npm ci
```

## 🧞 Commands

All commands are run from the root of the project, from a terminal:

| Command                   | Action                                           |
| :------------------------ | :----------------------------------------------- |
| `npm install`             | Installs dependencies                            |
| `npm run dev`             | Starts local dev server at `localhost:3000`      |
| `npm run build`           | Build your production site to `./dist/`          |
| `npm run preview`         | Preview your build locally, before deploying     |
| `npm run astro ...`       | Run CLI commands like `astro add`, `astro check` |
| `npm run astro -- --help` | Get help using the Astro CLI                     |

## How to deploy

Commit and push, it will deploy automatically via GitHub Pages.

## Download image from unsplash

```sh
curl -L "https://source.unsplash.com/1600x900/?team,collaborate" > src/img/team1.jpg
```

Copy image and resize as thumbnail:

```sh
cp img/mypost.jpg img/mypost-thumb.jpg
mogrify -resize 400x img/mypost-thumb.jpg
```

Install mogrify on Mac:

```sh
brew install imagemagick
```

Would be nice to be able to get the Author's name for attribution, but the URL above doesn't seem to provide it in response headers.