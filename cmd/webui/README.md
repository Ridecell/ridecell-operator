# Note:
This project has been stripped down to the point where buffalo no longer recognizes it as a buffalo project

# Building
We are using packr to build the binary as it eases the pain of including static files.

`packr2 build`

# Running binary locally
webui will panic if it does not find some required environment variables.

`HOST`
`GITHUB_KEY`
`GITHUB_SECRET`

`HOST=http://localhost GITHUB_KEY=REDACTED GITHUB_SECRET=GITHUB_SECRET ./webui`

# Force SSL
When deploying make sure to also set the environment variable `GO_ENV`.
If not set it will default to "development" and will allow http traffic.

Setting the variable to "production" will force https.
