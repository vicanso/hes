# errors

[![Build Status](https://img.shields.io/travis/vicanso/errors.svg?label=linux+build)](https://travis-ci.org/vicanso/errors)


Create a http error

# API

## HTTP Error

```go
he := &HTTPError{
  StatusCode: 500,
  Code: "cus-validate-fail",
  Category: "comon",
  Message: "error message",
  Exception: true,
  Extra: map[string]interface{}{
    "url": "http:///127.0.0.1/users/me",
  },
}
```

```go
he := New("error message")
```

```go
he := NewWithCaller("error message")
```

### Error

Get the description of http error

```go
he := &HTTPError{
  Message: "error message",
  Code: "cus-validate-fail",
  Category: "common",
}
// category=common, code=cus-validate-fail, message=error message
fmt.Println(he.Error())
```

### Format

Error format

```go
he := &HTTPError{
  Message: "error message",
  Code: "cus-validate-fail",
  Category: "common",
}
```

### SetCaller

Set the caller of error

```go
he := &HTTPError{
  Message: "error message",
  Code: "cus-validate-fail",
  Category: "common",
}
he.SetCaller(1)
```

### ToJSON

Error to json

```go
he := &HTTPError{
  Message: "error message",
  Code: "cus-validate-fail",
  Category: "common",
}
he.ToJSON()
```
