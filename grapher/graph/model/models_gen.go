// Code generated by github.com/99designs/gqlgen, DO NOT EDIT.

package model

type Answer struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

type Card struct {
	ID      string    `json:"id"`
	Title   string    `json:"title"`
	Answers []*Answer `json:"answers"`
}

type CreateAnswerInput struct {
	Text      string `json:"text"`
	IsCorrect bool   `json:"isCorrect"`
}

type CreateCardInput struct {
	Title   string               `json:"title"`
	Answers []*CreateAnswerInput `json:"answers"`
}

type CreateDeckInput struct {
	Title       string             `json:"title"`
	Description string             `json:"description"`
	IsPublic    bool               `json:"isPublic"`
	Cards       []*CreateCardInput `json:"cards"`
}

type CreateDeckResponse struct {
	Deck *Deck `json:"deck"`
}

type Deck struct {
	ID          string  `json:"id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Cards       []*Card `json:"cards"`
}

type DeleteDeckResponse struct {
	Success *bool `json:"success"`
}

type PageInfo struct {
	HasPreviousPage bool    `json:"hasPreviousPage"`
	HasNextPage     bool    `json:"hasNextPage"`
	StartCursor     *string `json:"startCursor"`
	EndCursor       *string `json:"endCursor"`
}

type PopularDeckEdge struct {
	Node   *Deck   `json:"node"`
	Cursor *string `json:"cursor"`
}

type PopularDecksConnection struct {
	Edges    []*PopularDeckEdge `json:"edges"`
	PageInfo *PageInfo          `json:"pageInfo"`
}

type Profile struct {
	ID          string  `json:"id"`
	Username    string  `json:"username"`
	DisplayName *string `json:"displayName"`
	Bio         *string `json:"bio"`
}
