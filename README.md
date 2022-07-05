# gin-gorm-rest

Simple golang gin & gorm REST endpoint handler generator.

Helps create simple API endpoints:

```
GET /resources
POST /resources
GET /resources/:resource
PUT /resources/:resource
DELETE /resources/:resource
```

Also has generators for associated endpoints, i.e. `GET /resources/:resource/children/:child`

Example simple endpoint: [owners.go](./example/owners.go)
Example associated endpoint: [owners_animals.go](./example/owners_animals.go)

Normal errors look like:
```json
{
    "message": "my error here"
}
```


Validation errors looks like this:
```json
{
    "message": "validation errors",
    "errors": {
        "name": "required",
        "<field>": "<validation error>"
    }
}
```

Developers
----------

Look at example directory for simple usage.
