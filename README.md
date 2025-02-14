## Update Commands
```sh
sudo systemctl stop wings
sudo curl -L -o /usr/local/bin/wings "https://github.com/RestoreMonarchy/PterodactylWings/releases/latest/download/wings_linux_$([[ "$(uname -m)" == "x86_64" ]] && echo "amd64" || echo "arm64")"
sudo chmod u+x /usr/local/bin/wings
sudo systemctl restart wings
```
