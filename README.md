# Toshokan

Toshokan(Japanese for library) is a flashcard backend system built with Go. It offers an API to interact with collections of flashcards named "decks".
Although in initial stages, it is under development together with Shisho(Japanese for librarian), a GUI client built with Rust and Iced.

## Config

Config files can be found under `env`. There are several configurable parameters like ports, hosts or user token expiry.


## Building & Setting Up

First build all services
```
$ make proto
$ make services
```

Then build and spin up the database
```
$ docker-compose up db
```

Run migrations
```
$ make migrations
```

And finally bring all the project up
```
$ docker-compose stop
$ docker-compose up
```

## Testing & Coverage

To run the tests
```
$ make test
```

To see test coverage
```
$ make coverage
```


## Examples with curl
```
// SignUp
curl -d '{"password": "secure-password", "username": "Peter", "bio": "Your neighbour", "nick": "Spiderman"}' -H "Content-Type: application/json" -X POST http://localhost:8080/signup -w "\n" -v
```
```
// LogIn
curl -d '{"password": "secure-password", "username": "Peter"}' -H "Content-Type: application/json" -X POST http://localhost:8080/login -w "\n" -v
```
```
// Create Deck
curl -d '
    {
        "title":"Golang", "description":"Learn about the Go programming language", 
        "cards":[
            {
                "title": "Which is the underlying data type of a slice in Go?",
                "possible_answers": [
                    {
                        "text": "Map","is_correct": false
                    },
                    {
                        "text": "Linked list","is_correct": false
                    },
                    {
                        "text": "Array","is_correct": true
                    }
                ]
            }
        ]
    }' -H "Content-Type: application/json" -H "Authorization: Bearer <token>" -X POST http://localhost:8080/decks/create -w "\n" -v
```

```
// Get Deck (use <deckUUID> returned by API in previous step)
curl -H "Content-Type: application/json" -H "Authorization: Bearer <token>" -X GET http://localhost:8080/decks/<deckUUID> -w "\n" -v
```
```
// Delete Deck
curl -H "Authorization: Bearer <token>" -X POST http://localhost:8080/delete/<deckUUID> -w "\n" -v
```

```
// Get Decks - returns only `Title` and `Description` for all decks
curl -H "Content-Type: application/json" -H "Authorization: Bearer <token>" -X GET http://localhost:8080/decks -w "\n" -v
```