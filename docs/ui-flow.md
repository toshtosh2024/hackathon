# UI/UX Design

## Concept

Next Market is a flea market app where AI helps both sellers and buyers.

- Sellers can generate item descriptions from rough notes.
- Buyers can ask AI whether an item matches their needs.
- Users can like, purchase, and message each other.

## Main Screens

1. Login / Register
2. Home / Item list
3. Item detail
4. Create item
5. Messages
6. My page

## User Flow

```text
Register/Login
  -> Home
    -> Item detail
      -> Ask AI
      -> Like
      -> Message seller
      -> Purchase
    -> Create item
      -> Generate description with OpenAI
      -> Publish item
    -> Messages
      -> Select conversation
      -> Send message
```

## Demo Flow

1. Create a seller account.
2. Open the create item screen.
3. Enter item title, price, category, and short notes.
4. Use OpenAI to generate a better description.
5. Publish the item.
6. Log in as a buyer.
7. Open the item detail screen.
8. Ask OpenAI a purchase question.
9. Like the item, send a DM, then purchase it.
