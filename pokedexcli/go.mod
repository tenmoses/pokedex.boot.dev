module github.com/tenmoses/pokedexcli

go 1.22.0

replace github.com/tenmoses/pokeapi v0.0.0 => ../pokeapi
replace github.com/tenmoses/pokecache v0.0.0 => ../pokecache

require (
	github.com/tenmoses/pokeapi v0.0.0
	github.com/tenmoses/pokecache v0.0.0
)
