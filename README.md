
# Let's Build

Source code for Simon's personal blog:

https://letsbuild.cloud

## Installation and pre-requisites

Install tailwind globally:

```sh
npm i -g tailwindcss
```

## How to build

Generate new CSS:

```sh
npx tailwindcss -i css/input.css -o css/output.css
```

## How to run locally

Use docker compose to run the image locally for testing purposes:

```sh
docker-compose up
docker-compose down
```

## How to deploy

Commit and push, it will deploy automatically via GitHub Pages.

## Download image from unsplash

```sh
curl -L "https://source.unsplash.com/1600x900/?team,collaborate" > img/team1.jpg
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