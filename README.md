# midus - Turn any function into a net/http handler

A Flexible ServeMux and HandlerFunc - Implement interfaces
to determine how function arguments, results, and errors are
mapped to the http request and response. Then write
functions instead of `http.Handlers` or
`http.HandlerFunc`'s.

## Arguments
Arguments must implement `midus.FromRequest`, which looks
like:

```go
type FromRequest interface {
	FromRequest(*http.Request) error
}
```

However, if (at most) one parameter does *not* implement this
interface, then that parameter will be decoded from json
instead, by using the standard encoding/json package
(supporting other types in the future is possible).

## Return Values
Return values must implement `midus.ToResponse`, which looks
like:

```go
type ToResponse interface {
	ToResponse(http.ResponseWriter) error
}
```

However, if (at most) one return does *not* implement this
interface, then that return value will be encoded to json
instead, by using the standard encoding/json package
(supporting other types in the future is possible).

## Errors
If a function returns an error (must be the last return
value), then the result will be a 500 internal server error
and no additional information about the error will be shown.
*However*, if the error also implements the
`midus.HTTPError` interface then the response code will be
obtained by calling `error.ResponseCode()`, and the response
body will be obtained from calling `error.Error()`. The
`midux.HTTPError` interface looks like:

```go
type HTTPError interface {
	error
	ResponseCode() int
}
```

## Isn't Reflection too slow?
Probabaly for some uses, *however* you can also run the
`midus` command line tool via `go generate` to use code
generation in place of reflection and  eliminate all
per-request reflection. (there will still be a small amount
of reflection during setup).

##TODO
- Make generate work with methods
- Add a tutorial
