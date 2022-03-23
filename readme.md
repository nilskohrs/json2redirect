# Json2Redirect

Json2Redirect is a middleware plugin for [Traefik](https://github.com/traefik/traefik) to redirect a request based on the response body.
To learn more about how JSON Pointers you can read the [RFC6901](https://datatracker.ietf.org/doc/html/rfc6901)

Content-Encoding is unsupported, this means only responses without the "Content-Encoding" header or of which the value "entity" can be processed. In any other case there will be a UnprocessableEntity 422 response code.
If the response body can't be parsed as a JSON this will return a UnsupportedMediaType 415 response code.
If the json path evaluation fails or the value can't be parsed as URL this will return a NotFound 404 response code.

## Configuration

### Static

```yaml
pilot:
  token: "xxxxx"

experimental:
  plugins:
    json2redirect:
      moduleName: "github.com/nilskohrs/json2redirect"
      version: "v0.0.1"
```

### Dynamic

```yaml
http:
  middlewares:
    json2redirect-foo:
      json2redirect:
        pointer: "/redirects/0/url"
```