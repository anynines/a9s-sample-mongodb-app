# a9s Sample Application for MongoDB

This is a sample app to check whether the a9s Data Service Automation for MongoDB service is working or not.

## Install, push and bind

Make sure you installed GO on your machine, [download this](https://golang.org/doc/install?download=go1.16.darwin-amd64.pkg) for mac.

Clone the repository
```
$ git clone https://github.com/anynines/a9s-mongodp-app
```

Create a service in Cloud Foundry. A MongoDB offering is needed.
```
$ cf create-service mongodb40 mongodb-single-small mymongodb
```

Push the app
```
$ cf push --no-start
```

Bind the app
```
$ cf bind-service mongodb-app mymongodb
```

And start
```
$ cf start mongodb-app
```

At last check the created url...


## Local test using Docker

To start it locally you should have Docker installed.
Afterwards just use the following command to create a MongoDB database and the mongodb application:

```
docker-compose up
```

If you made changes to the application itself, you can rebuild the docker image using

```
docker-compose build
```

## Remark

To bind the app to other MongoDB services than `mongodb40`, have a look at the `VCAPServices` struct.
