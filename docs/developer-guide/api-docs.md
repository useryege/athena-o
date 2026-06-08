# API Docs

You can find the Swagger docs by setting the path to `/swagger-ui` in your Athena UI. E.g. [http://localhost:8080/swagger-ui](http://localhost:8080/swagger-ui) or [http://localhost:4000/swagger-ui](http://localhost:4000/swagger-ui).

## Authorization

You'll need to authorize your API using a bearer token. To get a token:

```bash
$ curl -H "Content-Type: application/json" $ATHENA_SERVER/api/v1/session -d $'{"username":"admin","password":"password"}'
{"token":"eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpYXQiOjE1Njc4MTIzODcsImlzcyI6ImFyZ29jZCIsIm5iZiI6MTU2NzgxMjM4Nywic3ViIjoiYWRtaW4ifQ.ejyTgFxLhuY9mOBtKhcnvobg3QZXJ4_RusN_KIdVwao"} 
```

Then pass using the HTTP `Authorization` header, prefixing with `Bearer `:

```bash
$ curl $ATHENA_SERVER/api/v1/version -H "Authorization: Bearer $ATHENA_TOKEN"
{"Version":"...","BuildDate":"...","GitCommit":"...","GitTreeState":"...","GoVersion":"...","Compiler":"...","Platform":"..."}
```
