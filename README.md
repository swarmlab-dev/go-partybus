# 🚐🥳 go-partybus 

**go-partybus** is a webservice that can be used to create a temporary party bus. A bus is a simple passthrough between connected peers. 
This can be seen as an ephemeral rendez-vous meeting-room. 

## Features

- Service is available over HTTP/Websocket 🕸️🧦
- Server can host multiple party in parrallel 🚌🥳.
- ⚡ Sessions are created on demand and destroyed when all guests left the party
- Simple JSON based protocol allowing basic identification for guests 🤠
- Possibility to send a message to a subset of guest (broadcast 📢 / multicast / unicast)
- Events to keep track of how many guests are in the room 

