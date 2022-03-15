# Simple REST API Server

This API server provides endpoints to create, update, delete & list the metadata of `Application` objects in an in-memory datastore.

## To Start API Server
```$ git clone git@github.com:Fei-Guo/simple-apiserver.git```

```$ cd simple-apiserver```

```$ make```

```$ ./simple-apiserver``` [the default port is 8082 which can be overridden by using option --port]

## DataStore

The datastore supports storing multiple versions of application objects in the memory. For each object,
the revision history is recorded with an increasing `ResourceVersion` number. When deleting an object,
a new revision is added for the object with the `DeleteTimeStamp` field being set. A few notes:
- The db uses the `title` of each application as the key.
- With the support of saving object revisions in the db, it will be possible to support the `watch` mechanism
for the client (not implemented in this repo) to notify changes.
- To avoid the indefinite db growth, a simple compaction mechanism is implemented.
- To avoid the memory burst, a rate limiter is added in the apiserver.


## API
``````
type Maintainer struct {
	Name  string `yaml:"name"`
	Email string `yaml:"email"`
}

type Application struct {
	Title       string       `yaml:"title"`
	Version     string       `yaml:"version"`
	Maintainers []Maintainer `yaml:"maintainers,omitempty"`
	Company     string       `yaml:"company,omitempty"`
	Website     string       `yaml:"website,omitempty"`
	Source      string       `yaml:"source,omitempty"`
	License     string       `yaml:"license,omitempty"`
	Description string       `yaml:"description,omitempty"`

	// metadata
	CreateTimeStamp time.Time  `yaml:"createTimeStamp,omitempty"`
	DeleteTimeStamp *time.Time `yaml:"deleteTimeStamp,omitempty"`
	ResourceVersion string     `yaml:"resourceVersion,omitempty"`
}
``````

## Available API Endpoints

|  Method | API Endpoint  | Description |
|---|---|---|
|GET| /api/applications | Return a list of all applications with the latest version in response.|
|GET| /api/applications?company=XX&source=YY | Return a list of all matching applications with the latest version in response. Support querying by `company` and/or `source` for now.|
|GET| /api/applications/{title} | Return the latest version of the application `title` in response.|
|GET| /api/applications/{title}?dump | Return all versions of the application `title` in response.|
|POST| /api/applications | Add the applications in batches and return the results in response. |
|PUT| /api/applications | Update the applications in batches and return the results in response.|
|DELETE| /api/applications/{title} | Delete application 'title'|


## Test: Sample Curl commands

- Run API server

```shell
$ ./simple-apiserver --port=8082
```

- Create 4 objects specified in the [`sample/create.yaml`](./sample/create.yaml). Two will fail.

```shell
$ curl -X POST  -H "Content-Type:text/x-yaml" --data-binary "@sample/create.yaml" http://localhost:8082/api/applications
Err: application app3's version cannot be empty
Err: application app4's maintainer has wrong email fankgmail.com: mail: missing '@' or angle-addr
app1 has been created/updated successfully
app2 has been created/updated successfully
```

- Update two objects with the version specified in the [`sample/update.yaml`](./sample/update.yaml). One will fail since nothing is changed.

```shell
$ curl -X PUT  -H "Content-Type:text/x-yaml" --data-binary "@sample/update.yaml" http://localhost:8082/api/applications
Err: application app1 is up-to-date, abort update
app2 has been created/updated successfully
```

- List applications with the query `company=random2`. The latest revision is returned (resourceVersion 1).

```shell
$ curl -X GET -H "Content-Type:text/x-yaml" http://localhost:8082/api/applications\?company\=random2
- title: app2
  version: v0.0.1
  maintainers:
    - name: Bob
      email: bob@gmail.com
    - name: Jack
      email: Jack@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  resourceVersion: "1"
```

- Get application `app2` with full revision history. We can see that one maintainer has been changed from `Bob` to `Jack`.

```shell
$ curl -X GET -H "Content-Type:text/x-yaml" http://localhost:8082/api/applications/app2\?dump
- title: app2
  version: v0.0.1
  maintainers:
    - name: Alice
      email: alice@gmail.com
    - name: Bob
      email: bob@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  resourceVersion: "0"
- title: app2
  version: v0.0.1
  maintainers:
    - name: Bob
      email: bob@gmail.com
    - name: Jack
      email: Jack@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  resourceVersion: "1"
```

- Delete application `app2`. The next Get request will fail.

```shell
$ curl -X DELETE http://localhost:8082/api/applications/app2
app2 is deleted successfully
$ curl -X GET -H "Content-Type:text/x-yaml" http://localhost:8082/api/applications/app2
Err: application with title app2 has been deleted
```

- Dump the full revisions of `app2` again. The last revision has `DeleteTimeStamp` being set.

```shell
$ curl -X GET -H "Content-Type:text/x-yaml" http://localhost:8082/api/applications/app2\?dump
- title: app2
  version: v0.0.1
  maintainers:
    - name: Alice
      email: alice@gmail.com
    - name: Bob
      email: bob@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  resourceVersion: "0"
- title: app2
  version: v0.0.1
  maintainers:
    - name: Bob
      email: bob@gmail.com
    - name: Jack
      email: Jack@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  resourceVersion: "1"
- title: app2
  version: v0.0.1
  maintainers:
    - name: Bob
      email: bob@gmail.com
    - name: Jack
      email: Jack@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  deleteTimeStamp: 2022-03-15T13:08:03.851351-07:00
  resourceVersion: "2"

```

- The compaction is triggered every minute by default. The default number of revision retention is 2.
The apiserver will output the following log shortly.

```shell
Start compaction at 2022-03-15 13:08:16.951059 -0700 PDT m=+2040.024947043.
Compact app2 to the length 2.
```

- We can confirm that only the last two revisions are kept in the db.

```shell
$ curl -X GET -H "Content-Type:text/x-yaml" http://localhost:8082/api/applications/app2\?dump
- title: app2
  version: v0.0.1
  maintainers:
    - name: Bob
      email: bob@gmail.com
    - name: Jack
      email: Jack@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  resourceVersion: "1"
- title: app2
  version: v0.0.1
  maintainers:
    - name: Bob
      email: bob@gmail.com
    - name: Jack
      email: Jack@gmail.com
  company: random2
  website: https://app2.com
  source: https://github.com/random2/repo
  license: apache-2.2
  description: |
    ### blob of markdown
    More markdown1
  createTimeStamp: 2022-03-15T12:34:36.800709-07:00
  deleteTimeStamp: 2022-03-15T13:08:03.851351-07:00
  resourceVersion: "2"
```
