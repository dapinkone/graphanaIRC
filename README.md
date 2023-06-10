# graphanaIRC
IRC bot that sends alerts to IRC
Planned Features:

Bot stuff
- [ ] connect to IRC (SASL preferred)
- [ ] IRC command to set alert rate limit.
- [ ] IRC command to remove alert from list of concerns.
- [ ] Default alerts to being in "list of concerns"
- [ ] IRC command to add alert to list of concerns ( removing from "remove alert" blacklist )
- [ ] IRC command to temporarily mute bot for a period of time.
- [ ] possible format request: 10h10m30s etc

Graphana stuff
- [ ] Host simple web endpoint to receive JSON post requests ( to receive an alert from graphana )
- [ ] JSON parsing of alerts
- [ ] Rate Limiting: Determine if the alert has been reported within configurable time window. If not, report to IRC.
- [ ] Rate Limiting duration configurable per alert.

Other
- [ ] connect to postgresQL to store IRC bot settings
- [ ] take startup configs (IRC server, username, password) from a yaml file
- [ ] Dockerize
