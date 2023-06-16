# graphanaIRC
IRC bot that sends graphana alerts to IRC
Planned Features:

Bot stuff
- [X] connect to IRC (SASL preferred)
- IRC commands:
  - [ ] set alert rate limit.
  - [ ] mute alert [duration]
  - [ ] list alerts which are muted
  
  - [X] join / part channel
  - [X] quit command

- [ ] IRC command to temporarily mute bot for a period of time.
- [X] possible format request: 10h10m30s etc (supported up to hours. Days+ not supported)
- [ ] Authentication based on who has +o, +a, etc
- [X] Autojoin channels

Graphana stuff
- [X] Host simple web endpoint to receive JSON post requests ( to receive an alert from graphana )
- [X] JSON parsing of alerts
- [ ] Rate Limiting: Determine if the alert has been reported within configurable time window. If not, report to IRC.
- [ ] Rate Limiting duration configurable per alert.

Other
- [ ] connect to postgresQL to store IRC bot settings(blacklist)
- [X] take startup configs (IRC server, username, password) from a yaml file
- [ ] Dockerize


Notes:
Alerts should have:
- [ ] alert_name
- [ ] rate_limit ( eg, report only every 15m )
- [ ] muted_until: timestamp, stored as unix timestamp.
  This alert is ignored until [timestamp], and defaults to maxint or "forever"
  