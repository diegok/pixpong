# pixpong - Multiplayer Network Pong Game Design

## Overview

**pixpong** is a multiplayer terminal Pong game played over local network. Two teams compete to score points by getting the ball past the opposing team's defenders. Inspired by classic Pong but extended for team play with multiple players per side.

## Core Concept

### Team Structure
- Players split into two teams (left vs right) randomly at game start
- Each player occupies their own vertical column, evenly distributed across their team's half
- All paddles can move the full court height (up/down only)
- More players = more defensive columns the ball must pass through

### Example Layouts
```
1v1:    |P|          *          |P|

2v2:    |P|    |P|   *   |P|    |P|

3v3:    |P|  |P|  |P| * |P|  |P|  |P|
```

### Win Condition
- First team to reach the target score wins
- Target score configurable by host (default: 10 points)

### Court Size
- Determined by the smallest connected terminal (like pixwar)
- Ensures all players see the full court without clipping

## Ball Physics & Scoring

### Ball Behavior
- Ball represented by `⬤` (large filled circle)
- Bounce angle determined by hit position on paddle:
  - Center hit → ball returns straight
  - Edge hit → sharper angle
- Ball speed increases slightly with each paddle hit
- Speed cap scales with player count (more players = higher max speed)

### Collision Detection
- Simple physics - ball bounces off any paddle it touches
- Back-row players act as "second chance" defenders
- Ball bounces off top/bottom court walls

### Scoring
- Point awarded when ball crosses the goal line (left or right edge)
- After a goal:
  1. 2-second pause with score update
  2. Countdown displayed (3, 2, 1)
  3. Ball respawns at center
  4. Ball launches toward the team that just scored (counter-attack opportunity for scored-on team)

### Score Display
- Stadium-style scoreboard at center-top of screen
- Team colors shown with scores

## Players & Balancing

### Player Count
- Minimum: 2 players (1v1)
- Maximum: flexible (any number, auto-balanced between teams)
- Odd player count: one team gets an extra player

### Team Assignment
- Players join a pool in the lobby
- Teams randomly assigned when host starts the match
- No mid-game joining allowed

### Color Assignment
- Each player assigned a unique color (like pixwar)
- Colors: red, blue, green, yellow, purple, orange, teal, fuchsia

### Paddle Sizing
- Paddle size scales inversely with player count per side
- Fewer players = larger paddles
- More players = smaller paddles
- Keeps total defensive coverage roughly consistent

### Paddle Visuals
- Solid block character: `█`
- 1 character wide
- Height determined by player count scaling
- Colored per player

### Player Names
- Shown in lobby with assigned colors
- Hidden during gameplay (colored paddles only)
- Shown on end screen with final scores

## Lobby & Game Flow

### Server Startup
1. Host runs `pixpong --server`
2. Server displays available IP addresses for LAN sharing
3. Host waits in lobby for players to join
4. Lobby shows: player list, colors, configured settings

### Client Connection
1. Player runs `pixpong --join <ip:port>`
2. Client sends terminal size for court negotiation
3. Server validates terminal meets minimum size
4. Player appears in lobby with assigned color

### Game Start
1. Host presses Enter when 2+ players connected
2. Teams randomly assigned from player pool
3. Countdown: 3, 2, 1
4. Ball spawns at center, launches in random direction

### Mid-Game Rules
- No new players can join once match starts
- If any player disconnects:
  - Game ends immediately
  - All players return to lobby
  - Match is voided (no winner)

### Game End & Rematch
1. Final scores displayed with team results
2. Rematch voting phase begins
3. Players signal ready for rematch
4. If all players agree → new match with same settings (teams re-randomized)
5. If any player declines → return to lobby

## Command Line Interface

```
pixpong --server [options]     # Start as server/host
pixpong --join <ip:port>       # Join as client

Options:
  --port <N>      Server port (default: 5555)
  --name <name>   Player display name
  --points <N>    Points to win, server only (default: 10)
```

## Technical Architecture

### Technology Stack
- Language: Go
- Terminal UI: tcell
- Networking: TCP with gob encoding
- Architecture: Single binary, dual-mode (server/client)

### Server-Authoritative Model
- All game logic runs on server
- Server broadcasts complete game state at ~20 ticks/sec
- Clients send only input events (up/down movement)
- Ensures consistent state across all clients

### Message Types
| Message | Direction | Purpose |
|---------|-----------|---------|
| JoinRequest | Client → Server | Connection with terminal size |
| JoinResponse | Server → Client | Accept/reject with player ID |
| LobbyState | Server → All | Player list, settings, IPs |
| StartGame | Server → All | Triggers countdown |
| Countdown | Server → All | 3, 2, 1 display |
| GameState | Server → All | Ball, paddles, scores |
| PlayerInput | Client → Server | Up/down movement |
| GameOver | Server → All | Final scores |
| RematchReady | Client → Server | Ready signal |
| RematchState | Server → All | Ready status per player |

### Project Structure
```
cmd/pixpong/         - Entry point, IP detection
internal/
  app/               - Main event loop, state machine
  client/            - TCP client, message handling
  server/            - TCP server, game loop, broadcasting
  game/              - Ball physics, paddle logic, court
  protocol/          - Message types, gob codec
  ui/                - Terminal rendering, input handling
  config/            - CLI argument parsing
```

## Game State Details

### Ball State
- Position (x, y as floats for smooth movement)
- Velocity (vx, vy)
- Current speed multiplier

### Paddle State
- Player ID
- Column position (x, fixed per player)
- Vertical position (y, player controlled)
- Height (calculated from player count)
- Color

### Court State
- Width, height (from smallest terminal)
- Goal lines (x=0 for right team goal, x=width for left team goal)

### Match State
- Left team score
- Right team score
- Target score
- Game phase (lobby, countdown, playing, paused, game over, rematch)
