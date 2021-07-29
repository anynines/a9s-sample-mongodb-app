# a9s Sample Application for MongoDB

This is a sample app to check whether the a9s Data Service Automation for MongoDB
service is working or not.

# Table of Content

- [Install, Push and Bind](#install-push-and-bind)
- [Local test using Docker](#local-test-using-docker)
- [Remark](#remark)
## Install, Push and Bind

Clone the repository
```
$ git clone https://github.com/anynines/a9s-sample-mongodb-app.git
```

Create a service in Cloud Foundry. A MongoDB offering is needed.
```
$ cf create-service mongodb40 mongodb-single-small mymongodb
```

Before pushing the application verify that the service instance is ready to use.
If so, push the app with binding the service
```
$ cf push --var service=mymongodb
```

At last check the created url...


## Local test using Docker

To start it locally you should have Docker installed.
Afterwards just use the following command to create a MongoDB database and the
mongodb application:

```
docker-compose up
```

If you made changes to the application itself, you can rebuild the docker image
using

```
docker-compose build
```

## Remark

- To bind the app to other MongoDB services than `mongodb40`, have a look at the
  `VCAPServices` struct.
- To run the app locally, make sure you installed Go on your machine,
  [download this](https://golang.org/doc/install?download=go1.16.darwin-amd64.pkg) for macOS.

