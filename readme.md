

# Checkers Game on Cosmos SDK

This project demonstrates how to build a basic checkers game module on the Cosmos SDK. It includes creating a custom module, defining game mechanics, implementing message handling, and setting up queries.

## Prerequisites

- **Go**: [Install Go](https://golang.org/doc/install) if you haven't already.
- **Cosmos SDK**: Install and set up your Cosmos SDK environment by following the [Cosmos SDK installation guide](https://docs.cosmos.network/main/building).

## Project Setup

1. **Clone the repository**:
   ```bash
   git clone <your-repo-url>
   cd checkers-module
   ```

2. **Initialize a new Cosmos SDK Application**:
   ```bash
   starport scaffold chain github.com/username/checkers
   ```

3. **Start the Blockchain**:
   ```bash
   starport chain serve
   ```

## Step-by-Step Guide

### 1. Build the Module

Follow the [module-building guide](https://tutorials.cosmos.network/hands-on-exercise/0-native/2-build-module.html) to set up your custom `checkers` module.

- **Scaffold the Module**:
  ```bash
  starport scaffold module checkers --dep bank
  ```

- **Define Game Types**:
  Add types to handle game data and rules:
  ```bash
  starport scaffold type Game index playerBlack playerRed --module checkers
  ```

- **Update Module Logic**:
  Modify the game logic in `x/checkers/keeper/` and `x/checkers/types/` files to implement checkers rules.

### 2. Add Game Mechanics

Use the [add game guide](https://tutorials.cosmos.network/hands-on-exercise/0-native/3-add-game.html) to define the checkers game's data structure and mechanics.

- **Define Game Data**:
  Define the state variables for each game, like `playerBlack`, `playerRed`, and `gameState`.

- **Implement Game Logic**:
  Update the logic in the keeper functions to initialize and manage games. Implement basic game flow such as setting player moves, switching turns, and checking win conditions.

### 3. Add a Message

Add custom messages to interact with the module as per the [message guide](https://tutorials.cosmos.network/hands-on-exercise/0-native/4-add-message.html).

- **Create Messages**:
  Scaffold a message for creating a new game:
  ```bash
  starport scaffold message CreateGame playerBlack playerRed --module checkers
  ```

- **Handle the Message**:
  Implement the handler function in `x/checkers/keeper/msg_server_create_game.go` to create a game with the provided players and initialize the game state.

- **Register Message**:
  Register your message type in `x/checkers/module.go` to enable transactions for creating games.

### 4. Add a Query

Implement queries to retrieve game data as shown in the [query guide](https://tutorials.cosmos.network/hands-on-exercise/0-native/5-add-query.html).

- **Scaffold a Query**:
  Add queries to retrieve game information:
  ```bash
  starport scaffold query GetGame index --module checkers
  ```

- **Implement Query Logic**:
  Define query functions in `x/checkers/keeper/query_get_game.go` to retrieve game data based on game IDs.

- **Register the Query**:
  Register the query in `x/checkers/module.go` so users can interact with it via CLI or API.

### Usage Examples

1. **Creating a Game**:
   ```bash
   checkersd tx checkers create-game <black_player_address> <red_player_address> --from <your_wallet>
   ```

2. **Querying a Game**:
   ```bash
   checkersd query checkers get-game <game_id>
   ```

### Testing

1. **Run Tests**:
   Run unit tests to verify game logic:
   ```bash
   go test ./x/checkers/...
   ```

2. **CLI Testing**:
   Use CLI commands to manually test game creation and queries on the running blockchain.

### Cleanup

To reset the chain and start fresh:
```bash
checkersd unsafe-reset-all
```

### Additional Resources

- [Cosmos SDK Documentation](https://docs.cosmos.network/main/)
- [Cosmos SDK Tutorials](https://tutorials.cosmos.network/)

# Torram_Assignment
