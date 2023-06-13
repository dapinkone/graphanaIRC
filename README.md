# graphanaIRC
IRC bot that sends graphana alerts to IRC
Planned Features:

Bot stuff
- [X] connect to IRC (SASL preferred)
- IRC commands:
  - [ ] set alert rate limit.
  - [ ] add alert to an ignore blacklist
  - [ ] remove alert from blacklist.
  - [ ] list alerts in blacklist
  
  - [X] join / part channel
  - [X] quit command

- [ ] IRC command to temporarily mute bot for a period of time.
- [ ] possible format request: 10h10m30s etc
- [ ] Authentication based on who has +o, +a, etc
- [ ] Autojoin channels

Graphana stuff
- [ ] Host simple web endpoint to receive JSON post requests ( to receive an alert from graphana )
- [ ] JSON parsing of alerts
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
- [ ] Boolean: ignored_flag ( this alert is blacklisted, ignored indefinitely )
- [ ] muted_until: timestamp ( this alert is temp-ignored until [timestamp] )