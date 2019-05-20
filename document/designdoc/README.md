# Web-based Cloud Gaming Service Design Doc

Web-based Cloud Gaming Service contains multiple workers for gaming streaming and a coordinator (Overlord) for distributing traffic and setup connection.

## Worker 

Worker is responsible for streaming game to frontend 
![worker](../img/worker.png)

## Overlord

Overlord is in charge of picking the most suitable workers for a user. Everytime a user connects to overlord, it will collect all the metric from all workers, i.e free CPU resources and latency from  worker to user. Overlord will decide the best candidate based on the metric and setup peer-to-peer connection between worker and user based on WebRTC protocol

![Architecture](../img/overlord.png)
