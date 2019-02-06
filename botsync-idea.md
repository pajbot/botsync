
                +---------------------------+
                | KKona                     |
                | pubsub server             |
                |                           |
                |                           |
                |                           |
                |                           |
                | upon user state change    |
                | notify all pubsub clients |
                +---------------------------+
                   ^                 ^
                   |                 |
                   |                 |
                   |                 |
                   V                 V
    +---------------------+       +---------------------+
    |                     |       |                     |
    |   pubsub client     |       |   pubsub client     |
    |                     |       |                     |
    | pajbot2             |       | pajbot              |
    | pajbot #pajlada     |       | snusbot #forsen     |
    |                     |       |                     |
    |                     |       |                     |
    |                     |       |                     |
    +---------------------+       +---------------------+




pajlada #forsen: !afk shower
snusbot reads !afk command
snusbot tells KKona that user pajlada has gone afk with the message "shower"
interested clients, such as snusbot in #forsen, can choose to send a message with that update: "pajlada has gone afk: 'shower'"
