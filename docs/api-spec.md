# API Spec

Base URL: `/api`

## Auth

### POST /auth/register

Request:

```json
{
  "name": "Toshi",
  "email": "toshi@example.com",
  "password": "password"
}
```

Response:

```json
{
  "token": "jwt-like-token",
  "user": {
    "id": 1,
    "name": "Toshi",
    "email": "toshi@example.com",
    "role": "user"
  }
}
```

### POST /auth/login

Request:

```json
{
  "email": "toshi@example.com",
  "password": "password"
}
```

## Items

### GET /items

Response:

```json
{
  "items": [
    {
      "id": 1,
      "sellerId": 1,
      "sellerName": "Toshi",
      "title": "Vintage Jacket",
      "description": "Clean jacket in good condition.",
      "category": "fashion",
      "price": 7800,
      "status": "active",
      "imageUrl": "",
      "likeCount": 2,
      "createdAt": "2026-06-17T10:00:00Z"
    }
  ]
}
```

### POST /items

Authorization: `Bearer <token>`

Request:

```json
{
  "title": "Vintage Jacket",
  "description": "Clean jacket in good condition.",
  "category": "fashion",
  "price": 7800,
  "imageUrl": ""
}
```

### GET /items/{id}

Returns one item.

### POST /items/{id}/like

Authorization: `Bearer <token>`

Toggles a like and returns the current state.

### POST /items/{id}/purchase

Authorization: `Bearer <token>`

Marks an active item as sold.

## Messages

### POST /conversations

Authorization: `Bearer <token>`

Request:

```json
{
  "itemId": 1,
  "sellerId": 1
}
```

### GET /conversations

Authorization: `Bearer <token>`

Returns conversations for the current user.

### GET /conversations/{id}/messages

Authorization: `Bearer <token>`

Returns messages in a conversation.

### POST /conversations/{id}/messages

Authorization: `Bearer <token>`

Request:

```json
{
  "body": "Is this still available?"
}
```

## AI

### POST /ai/generate-description

Authorization: `Bearer <token>`

Request:

```json
{
  "title": "Vintage Jacket",
  "category": "fashion",
  "condition": "Good",
  "notes": "Bought in Shimokitazawa. No visible stains."
}
```

### POST /ai/ask

Authorization: `Bearer <token>`

Request:

```json
{
  "itemId": 1,
  "question": "Is this useful for a rainy day?"
}
```
