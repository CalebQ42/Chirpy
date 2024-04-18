# Chirpy

A twitter clone backend.

A boot.dev

## API

> GET /admin/metrics/

HTML page with how many times /app pages have been viewed

> GET /api/healthz

Returns "OK"

> /api/reset

Reset the page view count

> POST /api/chirps

Post a chirp.

Request Header:

```json
{
    "Authorization": "Bearer <access token>"
}
```

Request Body:

```json
{
    "body": "Chirp text"
}
```

Return:

```json
{
    "body": "Chirp text with bad words replaced with ****",
    "id": 0, // chirp ID
    "author": 0 // author's user ID,

}
```

> GET /api/chirps?author_id=\<author id\>&sort=\<asc or desc\>

author_id and sort are optional.

Returns a list of chirps, optionally specifying the author and sort order.

Return:

```json
[
    {
        "body": "Chirp text with bad words replaced with ****",
        "id": 0, // chirp ID
        "author": 0 // author's user ID,
    },
    ...
]
```

> GET /api/chirps/{chirpID}

Get a specific Chirp by ID

Return:

```json
{
    "body": "Chirp text with bad words replaced with ****",
    "id": 0, // chirp ID
    "author": 0 // author's user ID,

}
```
