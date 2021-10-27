# guardian-crossworder

An app which allows you to do Grauniad crosswords over the web with your friends.
Which can also be easily be deployed to Heroku.

## Running Locally

Make sure you have [Go](http://golang.org/doc/install) version 1.17 or newer

```sh
$ go run .
```
Your app should now be running on http://localhost:5000/quick/16060

You can change the Grauniad crossword type and number as desired

Open the same link in another browser and solve to see the web sockets in action



## Deploying to Heroku

Make sure you have [Heroku CLI](https://devcenter.heroku.com/articles/heroku-cli) installed.

```sh
$ heroku create
$ git push heroku main
```

or

[![Deploy](https://www.herokucdn.com/deploy/button.svg)](https://heroku.com/deploy)

