// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type Answer struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

type AnswerCardsInput struct {
	AnswerIDs []string `json:"answerIDs"`
}

type AnswerCardsResponse struct {
	AnswerIDs []string `json:"answerIDs"`
}

type Card struct {
	ID          string    `json:"id"`
	Title       string    `json:"title"`
	Answers     []*Answer `json:"answers,omitempty"`
	Explanation *string   `json:"explanation,omitempty"`
}

type CardsInput struct {
	DeckID   string `json:"deckID"`
	MaxCards int    `json:"maxCards"`
}

type CreateAnswerInput struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

type CreateCardInput struct {
	Title       string               `json:"title"`
	Answers     []*CreateAnswerInput `json:"answers"`
	Explanation *string              `json:"explanation,omitempty"`
}

type CreateDeckCardInput struct {
	Card   *CreateCardInput `json:"card"`
	DeckID string           `json:"deckID"`
}

type CreateDeckCardResponse struct {
	Success bool `json:"success"`
}

type CreateDeckInput struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	IsPublic    bool               `json:"isPublic"`
	Cards       []*CreateCardInput `json:"cards"`
}

type CreateDeckResponse struct {
	Deck *Deck `json:"deck,omitempty"`
}

type Deck struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Cards       []*Card `json:"cards,omitempty"`
}

type DeleteDeckResponse struct {
	Success *bool `json:"success,omitempty"`
}

type PageInfo struct {
	HasPreviousPage bool    `json:"hasPreviousPage"`
	HasNextPage     bool    `json:"hasNextPage"`
	StartCursor     *string `json:"startCursor,omitempty"`
	EndCursor       *string `json:"endCursor,omitempty"`
}

type PopularDeckEdge struct {
	Node   *Deck   `json:"node,omitempty"`
	Cursor *string `json:"cursor,omitempty"`
}

type PopularDecksConnection struct {
	Edges    []*PopularDeckEdge `json:"edges,omitempty"`
	PageInfo *PageInfo          `json:"pageInfo"`
}

type Profile struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"displayName,omitempty"`
	Bio         *string `json:"bio,omitempty"`
}
