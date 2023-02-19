# Feature: Creating a Room

## Encryption 

TODO

## Description

The user must post a message to the courier-rest server indicating the room's parameters (name, etc). Upon success, the 
server will return a URL which can be used to join the room. 

## Flow

Client

# Feature: Joining a Room

## Encryption

TODO

## Description

Room creation will generate a URL which is sharable and can be used any number of times. A user will POST the URL. (TODO private rooms).
Once the room has been joined, the user will see the option to view the chat in the UI. POSTing the room URL to join will
add the user to the list of users which should be recipients of new messages from courier.

