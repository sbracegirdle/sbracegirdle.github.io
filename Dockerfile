# Jekyll image base, install jekyll-feed and serve
FROM jekyll/jekyll:3.8.5
RUN gem install jekyll-feed
CMD ["jekyll", "serve"]
