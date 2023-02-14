# GroupMe-Clone (Working Name)

This is an attempt at re-creating the GroupMe app. Although the goal is to create a chat service, the aim of
this project is to help me gain experience working with websockets and encryption.

## Related Components

* This Repo - Websocket server
* ? - Web Client
* ? - iOS/Android Client

## Features Implemented

None

## Wishlist

* Real-time messaging between users
* Group chats with thousands of active users
* DMs
* Multimedia messages (images, videos, audio)

## Architecture

Straightforward client-server architecture. Each client holds a websocket connection to the server which allows the
server to push messages as soon as they are processed.

### Client -> Websocket Server -> Postgres Database

* Send messages to a chat room
    * Persist message in database
    * Push message to devices

### Client -> Rest API

* Join rooms
    * Private Rooms? Password-protected rooms?
* Update profile
    * Name, picture
* Add contacts, "friends"
    * Is this strictly necessary? Could be an allowlist of users who can initiate messages or view profile, unless
      already in a group

Websocket Server -> Client:

* Push new chat messages to device

### Clustering

A single server cannot scale infinitely. Using multiple instances of a server is a requirement if availability is a 
priority.

Consider a deployment on a kubernetes instance. The websocket servers should be able to scale up and down as needed.