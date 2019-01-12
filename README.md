# Piq
A command line tool to manage Bitmain Bitcoin Miners

***Developed on go 1.9***

### Installing

Clone repo:

` git clone git@github.com:jsmootiv/piq `

Install Glide: https://github.com/Masterminds/glide

- Install dependencies:

```bash
mkdir ~/.piq/
cp config.json.template ~/.piq/config.json  # Copy config
vim ~/.piq/config.json                      # Edit config
glide install                               # Install dependencies
go install piq.go                           # Installs app to your path
piq help
```

### Examples
TODO:

### Supported Pools
- Slushpool


### Command support

````
Usage:
  app [command]

Available Commands:
  help        Help about any command
  kill        Kills worker
  reboot      reboots worker
  stats       Pulls stats from worker or pool
```

## License

This project is licensed under the MIT License - see the [LICENSE.md](LICENSE.md) file for details
