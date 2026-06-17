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

Query:

- `q`: title and description keyword
- `category`: exact category
- `min_price`: minimum price
- `max_price`: maximum price

Response:

```json
{
  "items": [
    {
      "id": 1,
      "sellerId": 1,
      "sellerName": "Toshi",
      "sellerRatingAvg": 4.8,
      "sellerReviewCount": 12,
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

The backend runs an AI safety check before creating the item. If the item is likely prohibited, the response is `422` with a `review` object.

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

### GET /my/items

Authorization: `Bearer <token>`

Returns all items created by the current user, including hidden items.

### POST /upload

Authorization: `Bearer <token>`

Issues a Cloud Storage signed URL for direct image upload.

Request:

```json
{
  "filename": "bag.jpg",
  "contentType": "image/jpeg",
  "purpose": "item"
}
```

`purpose` is optional. Use `item` for listing images and `avatar` for profile images. Objects are stored under matching Cloud Storage prefixes.
Set `visibility` to `private` for non-public uploads such as profile photos.
When GCS environment variables are not set in local development, the backend falls back to `PUT /api/local-upload` and serves files from `/uploads`.

Response:

```json
{
  "uploadUrl": "https://storage.googleapis.com/...",
  "publicUrl": "https://storage.googleapis.com/nextmarket/items/...",
  "objectPath": "gcs://nextmarket/avatars/...",
  "method": "PUT",
  "contentType": "image/jpeg"
}
```

### GET /items/{id}

Returns one item.

### GET /items/{id}/ai-scene

Authorization: `Bearer <token>`

Returns the latest personalized AI usage-scene image for the current user and item.

### POST /items/{id}/ai-scene

Authorization: `Bearer <token>`

Generates a personalized usage-scene image using the current user's private profile photo and the item image.

### POST /items/{id}/cancel

Authorization: `Bearer <token>`

Hides an active listing owned by the current user.

### POST /items/{id}/like

Authorization: `Bearer <token>`

Toggles a like and returns the current state.

### POST /items/{id}/purchase

Authorization: `Bearer <token>`

Marks an active item as sold.

### GET /items/{id}/reviews

Returns reviews associated with a completed transaction for the item.

### POST /items/{id}/reviews

Authorization: `Bearer <token>`

Creates one review for the completed transaction. Only the buyer or seller can review the counterpart, and each side can review once.

Request:

```json
{
  "rating": 5,
  "comment": "Smooth and reliable transaction."
}
```

### GET /users/{id}/reviews

Returns reviews received by the user.

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
Each conversation also includes item summary fields and counterpart profile info for the DM view.

### GET /conversations/{id}/messages

Authorization: `Bearer <token>`

Returns messages in a conversation.

## Profile

### POST /profile

Authorization: `Bearer <token>`

Request:

```json
{
  "name": "Toshi",
  "avatarPath": "gcs://nextmarket/avatars/..."
}
```

### POST /conversations/{id}/messages

Authorization: `Bearer <token>`

Request:

```json
{
  "body": "Is this still available?"
}
```

## AI

### POST /ai/check-item

Authorization: `Bearer <token>`

Checks whether an item may be prohibited before listing.

Request:

```json
{
  "title": "Vintage Jacket",
  "description": "Clean jacket in good condition.",
  "category": "fashion",
  "condition": "used"
}
```

Response:

```json
{
  "review": {
    "prohibited": false,
    "riskLevel": "low",
    "reasons": ["No prohibited signals were found."],
    "blockedKeywords": []
  }
}
```

### POST /ai/suggest-price

Authorization: `Bearer <token>`

Suggests a realistic listing price in JPY.

Request:

```json
{
  "title": "Vintage Jacket",
  "category": "fashion",
  "condition": "good",
  "notes": "Clean and lightly used."
}
```

Response:

```json
{
  "suggestion": {
    "price": 7800,
    "minPrice": 6200,
    "maxPrice": 9300,
    "reason": "Similar used jackets sell in this range.",
    "signals": ["fashion", "good condition"]
  }
}
```

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
