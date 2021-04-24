# API

### Coordinator

HTTP: None

Packet:
    id 
    t
    payload

WebSocket:
 - The initial request should be wss://coordinator-address?room_id=id&zone=zone
   room_id (string) optional -- connect to the existing room
   zone (string) optional -- select new worker from the specified region (i.e. eu, us ...)


 - / init_webrt / 
 - / offer /
