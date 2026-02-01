# pixpong

A multiplayer terminal Pong game written in Go. Play with friends over your local network!

```
  3  -  5
First to 10 wins

     |              ⬤                    |
     |                                   |
  █  |                                   |  █
  █  |                                   |  █
  █  |                                   |  █
     |                                   |
     |                                   |  █
     |                                   |  █
                                         |  █
```

## Features

- **Multiplayer over LAN** - Host a game and have friends join from their terminals
- **Team-based gameplay** - Players are randomly split into left and right teams
- **Dynamic scaling** - Paddle size adjusts based on number of players per team
- **Ball physics** - Bounce angle depends on where the ball hits the paddle
- **Speed escalation** - Ball speeds up with each hit until someone scores
- **Configurable** - Set custom points-to-win
- **Rematch system** - Quick rematch voting after each game

## Installation

### From source

```bash
git clone https://github.com/diegok/pixpong.git
cd pixpong
go build ./cmd/pixpong
```

### Using go install

```bash
go install github.com/diegok/pixpong/cmd/pixpong@latest
```

## How to Play

### Start a server

```bash
./pixpong --server --name YourName
```

The server will display IP addresses that others can use to connect:

```
Starting PixPong server on port 5555
Players can connect using:

  pixpong --join 192.168.1.100:5555
  pixpong --join localhost:5555  (same machine)
```

### Join a game

```bash
./pixpong --join <host-ip>:5555 --name YourName
```

### Start the match

Once at least 2 players have joined, the host presses **Enter** to start.

## Controls

| Key | Action |
|-----|--------|
| `W` / `↑` | Move paddle up |
| `S` / `↓` | Move paddle down |
| `Enter` | Start game / Ready for rematch |
| `Q` / `Esc` | Quit |

## Command Line Options

```
Usage:
  pixpong --server [options]       Start a game server
  pixpong --join <address>         Join a game server

Options:
  --port <port>       Server port (default: 5555)
  --name <name>       Player name
  --points <n>        Points to win (default: 10)

Examples:
  pixpong --server --name Host
  pixpong --join 192.168.1.100 --name Player2
  pixpong --join localhost:5555 --name TestPlayer
```

## Game Rules

- Players are randomly assigned to left or right team when the game starts
- Each team defends their goal (left or right edge)
- Score a point by getting the ball past the opposing team's defenders
- Ball bounces off top/bottom walls and paddles
- Hit the ball with the edge of your paddle for sharper angles
- Ball speeds up with each paddle hit (capped based on player count)
- First team to reach the target score wins
- If any player disconnects, the game ends

## Requirements

- Go 1.21 or later
- Terminal with Unicode support
- Minimum terminal size: 40x20

## License

MIT
