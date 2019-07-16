# Transkard balance checker
![Example](https://user-images.githubusercontent.com/5345489/61315442-6ee13080-a807-11e9-81b0-d685df08ca49.png)

Works only for transport cards issued in Kazan, Russia (http://www.transkart.ru/)

# Requirements
- Telegram Bot
- PostgreSQL

# Example config
```yaml
Version: 1.0
LogLevel: info

Telegram:
  Token: "token from @BotFather"
  
DB:
  Host: localhost
  Port: 5432
  User: tc
  Password: tc1234
  Name: tc
  SSL: false
```
