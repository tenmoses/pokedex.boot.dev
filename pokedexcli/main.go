package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"time"

	"github.com/tenmoses/pokeapi"
	"github.com/tenmoses/pokecache"
)

func main() {
	fmt.Println("pokedex")

	readCh := make(chan string)
	defer close(readCh)

	go readFromCli(readCh)

	commands := getCommands()

	conf := config{
		Next:     0,
		Previous: 0,
	}

	cache := pokecache.NewCache(6000 * time.Millisecond)
	pokedex := make(map[string]pokeapi.PokemonToCatch)

	for commandLine := range readCh {
		commandName, args := parseCommand(commandLine)
		command, ok := commands[commandName]

		if ok {
			switch command.callback {
			case "commandHelp":
				commandHelp()
			case "commandMap":
				commandMap(&conf, cache)
			case "commandMapB":
				commandMapB(&conf, cache)
			case "commandExplore":
				if len(args) > 0 {
					commandExplore(args[0], cache)
				} else {
					fmt.Println("No location area name specified")
				}
			case "commandCatch":
				if len(args) > 0 {
					commandCatch(args[0], cache, pokedex)
				} else {
					fmt.Println("No location area name specified")
				}
			case "commandInspect":
				if len(args) > 0 {
					commandInspect(args[0], pokedex)
				} else {
					fmt.Println("No pokemon name specified")
				}
			case "commandPokedex":
				commandPokedex(pokedex)
			case "commandExit":
				return
			default:
				fmt.Println("No callback function found")
			}
		}
	}
}

func readFromCli(ch chan string) {
	scanner := bufio.NewScanner(os.Stdin)

	ok := true

	for ok {
		scanner.Scan()
		err := scanner.Err()
		if err != nil {
			ok = false
		}

		ch <- scanner.Text()
	}
}

func parseCommand(line string) (string, []string) {
	parts := strings.Split(line, " ")

	return parts[0], parts[1:]
}

type cliCommand struct {
	name        string
	description string
	callback    string
}

func getCommands() map[string]cliCommand {
	return map[string]cliCommand{
		"help": {
			name:        "help",
			description: "Displays a help message",
			callback:    "commandHelp",
		},
		"exit": {
			name:        "exit",
			description: "Exit the Pokedex",
			callback:    "commandExit",
		},
		"map": {
			name:        "map",
			description: "Displays the next 20 names of location areas in the Pokemon world",
			callback:    "commandMap",
		},
		"mapb": {
			name:        "mapb",
			description: "Displays previous 20 locations",
			callback:    "commandMapB",
		},
		"explore": {
			name:        "explore",
			description: "List of all the Pokémon in a given area",
			callback:    "commandExplore",
		},
		"catch": {
			name:        "catch",
			description: "Catching Pokemon adds them to the user's Pokedex. It takes the name of a Pokemon as an argument",
			callback:    "commandCatch",
		},
		"inspect": {
			name:        "inspect",
			description: "Takes the name of a Pokemon as an argument. Print the name, height, weight, stats and type(s) of the Pokemon",
			callback:    "commandInspect",
		},
		"pokedex": {
			name:        "pokedex",
			description: "Print a list of all the names of the Pokemon the user has caught",
			callback:    "commandPokedex",
		},
	}
}

func commandHelp() error {
	text := "\nWelcome to the Pokedex!\nUsage:\n\n"
	commandsText, err := getCommandsText(getCommands())

	if err != nil {
		return err
	}

	fmt.Println(text, commandsText)

	return nil
}

func getCommandsText(commands map[string]cliCommand) (string, error) {
	commandsText := ""

	if len(commands) == 0 {
		return commandsText, errors.New("no commands to display")
	}

	for _, command := range commands {
		commandsText += fmt.Sprintf("%s: %s\n", command.name, command.description)
	}

	return commandsText, nil
}

func commandMap(conf *config, cache pokecache.Cache) error {
	offset := conf.Next + 1

	toPrint, err := getNamesPage(offset, cache)

	if err == nil {
		fmt.Print(toPrint)

		conf.Previous = conf.Next
		conf.Next = conf.Next + 20
	} else {
		fmt.Print(err)
	}

	return nil
}

func commandMapB(conf *config, cache pokecache.Cache) error {
	if conf.Previous == 0 {
		fmt.Println("No previous")
	} else {
		//Prepare offset but not affect config till data is fetched
		offset := conf.Previous - 20 + 1

		toPrint, err := getNamesPage(offset, cache)

		if err == nil {
			fmt.Print(toPrint)

			conf.Next = conf.Previous
			conf.Previous = conf.Previous - 20
		} else {
			fmt.Print(err)
		}
	}

	return nil
}

func commandPokedex(pokedex map[string]pokeapi.PokemonToCatch) error {
	if len(pokedex) > 0 {
		fmt.Println("Your Pokedex:")

		for pokemonName := range pokedex {
			fmt.Printf("- %s\n", pokemonName)
		}

		return nil
	} else {
		fmt.Println("You caught no pokemons yet")
	}

	return nil
}

func commandInspect(name string, pokedex map[string]pokeapi.PokemonToCatch) error {
	pokemon, ok := pokedex[name]

	if !ok {
		fmt.Println("you have not caught that pokemon")
		return nil
	}

	fmt.Printf("Name: %s\n", pokemon.Name)
	fmt.Printf("Height: %v\n", pokemon.Height)
	fmt.Printf("Weight: %v\n", pokemon.Weight)
	fmt.Print("Stats:\n")

	for name, value := range pokemon.Stats {
		fmt.Printf("- %s: %v\n", name, value)
	}

	fmt.Print("Types:\n")

	for _, pType := range pokemon.Types {
		fmt.Printf("- %s\n", pType)
	}

	return nil
}

func commandCatch(name string, cache pokecache.Cache, pokedex map[string]pokeapi.PokemonToCatch) error {
	fmt.Printf("Throwing a Pokeball at %s...\n", name)

	pokemon, err := getPokemon(name, cache)

	if err != nil {
		fmt.Print(err)
		return nil
	}

	catched := tryToCatch(pokemon.BaseExperience)

	if catched {
		fmt.Printf("%s was caught!\n", name)
		pokedex[name] = pokemon
	} else {
		fmt.Printf("%s escaped\n", name)
	}

	return nil
}

func tryToCatch(baseExp int) bool {
	scale, difficulty, adjust := 1000, 1, 10
	catchChance := scale / ((difficulty * baseExp) + adjust)

	diceThrow := rand.Intn(10)

	return diceThrow <= catchChance
}

func getPokemon(name string, cache pokecache.Cache) (pokeapi.PokemonToCatch, error) {
	cached, ok := cache.Get(name)

	if ok {
		pokemonToCatch := pokeapi.PokemonToCatch{}

		json.Unmarshal(cached, &pokemonToCatch)
		return pokemonToCatch, nil
	} else {
		pokemonToCatch, err := pokeapi.GetPokemonToCatch(name)

		if err != nil {
			return pokemonToCatch, err
		} else {
			toCache, err := json.Marshal(pokemonToCatch)

			if err != nil {
				return pokemonToCatch, err
			}

			cache.Add(name, toCache)

			return pokemonToCatch, nil
		}
	}
}

func commandExplore(locationName string, cache pokecache.Cache) error {
	fmt.Printf("Exploring %s...\n", locationName)

	toPrint, err := getPokemonsInArea(locationName, cache)

	if err == nil {
		fmt.Println("Found Pokemon:")
		fmt.Print(toPrint)
	} else {
		fmt.Print(err)
	}

	return nil
}

func getPokemonsInArea(locationName string, cache pokecache.Cache) (string, error) {
	cacheKey := fmt.Sprintf("pokemonsInArea_%s", locationName)

	cached, ok := cache.Get(cacheKey)

	if ok {
		return fmt.Sprint(cached), nil
	} else {
		names, err := pokeapi.GetPokemonsInArea(locationName)

		if err != nil {
			return "", err
		} else {
			namesLine := ""
			for _, name := range names {
				namesLine += fmt.Sprintf("%s\n", name)
			}
			cache.Add(cacheKey, []byte(namesLine))

			return namesLine, nil
		}
	}
}

func getNamesPage(offset int, cache pokecache.Cache) (string, error) {
	cacheKey := fmt.Sprintf("page%d-%d", offset, offset+20)

	cached, ok := cache.Get(cacheKey)

	if ok {
		return fmt.Sprint(cached), nil
	} else {
		names, err := pokeapi.GetLocationAreaNames(20, offset)

		if err != nil {
			return "", err
		} else {
			namesLine := ""
			for _, name := range names {
				namesLine += fmt.Sprintf("%s\n", name)
			}
			cache.Add(cacheKey, []byte(namesLine))

			return namesLine, nil
		}
	}
}

type config struct {
	Next     int
	Previous int
}
