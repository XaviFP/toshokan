package graph

// This file will be automatically regenerated based on the schema, any resolver implementations
// will be copied through when generating and any unknown code will be moved to the end.

import (
	"context"

	v1 "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/grapher/graph/generated"
	"github.com/XaviFP/toshokan/grapher/graph/model"
	"github.com/juju/errors"
)

// CreateDeck is the resolver for the createDeck field.
func (r *mutationResolver) CreateDeck(ctx context.Context, input model.CreateDeckInput) (*model.CreateDeckResponse, error) {
	res, err := r.DeckClient.CreateDeck(ctx, &v1.CreateDeckRequest{
		Deck: &v1.Deck{
			Title:       input.Title,
			Description: input.Description,
			Cards:       cardsFromInput(input.Cards),
			AuthorId:    r.getUserID(ctx),
		},
	})

	if err != nil {
		return nil, errors.Trace(err)
	}

	out := &model.Deck{
		ID:          res.Deck.Id,
		Title:       res.Deck.Title,
		Description: res.Deck.Description,
		Cards:       cardsToModel(res.Deck.Cards),
	}

	return &model.CreateDeckResponse{Deck: out}, errors.Trace(err)
}

// DeleteDeck is the resolver for the deleteDeck field.
func (r *mutationResolver) DeleteDeck(ctx context.Context, id string) (*model.DeleteDeckResponse, error) {
	_, err := r.DeckClient.DeleteDeck(ctx, &v1.DeleteDeckRequest{Id: id})
	if err != nil {
		return nil, errors.Trace(err)
	}

	success := true
	return &model.DeleteDeckResponse{Success: &success}, nil
}

// Deck is the resolver for the deck field.
func (r *queryResolver) Deck(ctx context.Context, id string) (*model.Deck, error) {
	res, err := r.DeckClient.GetDeck(ctx, &v1.GetDeckRequest{DeckId: id, UserId: r.getUserID(ctx)})
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &model.Deck{
		ID:          res.Deck.Id,
		Title:       res.Deck.Title,
		Description: res.Deck.Description,
		Cards:       cardsToModel(res.Deck.Cards),
	}, nil
}

// PopularDecks is the resolver for the popularDecks field.
func (r *queryResolver) PopularDecks(ctx context.Context, first *int, after *string, last *int, before *string) (*model.PopularDecksConnection, error) {
	res, err := r.DeckClient.GetPopularDecks(ctx, &v1.GetPopularDecksRequest{
		UserId:     r.getUserID(ctx),
		Pagination: paginationFromInput(first, after, last, before)})
	if err != nil {
		return nil, errors.Trace(err)
	}

	out, err := r.connectionToModel(ctx, res.Connection)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return out, nil
}

func paginationFromInput(first *int, after *string, last *int, before *string) *v1.Pagination {
	var out v1.Pagination
	if first != nil {
		out.First = int64(*first)
	}

	if after != nil {
		out.After = *after
	}

	if last != nil {
		out.Last = int64(*last)
	}

	if before != nil {
		out.Before = *before
	}

	return &out
}

func (r *queryResolver) connectionToModel(ctx context.Context, conn *v1.PopularDecksConnection) (*model.PopularDecksConnection, error) {
	var edges []*model.PopularDeckEdge

	for _, e := range conn.Edges {
		deck, err := r.DeckLoader.Load(ctx, e.DeckId)
		if err != nil {
			return nil, errors.Trace(err)
		}

		edges = append(edges, &model.PopularDeckEdge{
			Node:   deckToModel(deck.(*v1.Deck)),
			Cursor: &e.Cursor,
		})
	}

	return &model.PopularDecksConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasPreviousPage: conn.PageInfo.HasPreviousPage,
			HasNextPage:     conn.PageInfo.HasNextPage,
			StartCursor:     &conn.PageInfo.StartCursor,
			EndCursor:       &conn.PageInfo.EndCursor,
		},
	}, nil
}

// Mutation returns generated.MutationResolver implementation.
func (r *Resolver) Mutation() generated.MutationResolver { return &mutationResolver{r} }

// Query returns generated.QueryResolver implementation.
func (r *Resolver) Query() generated.QueryResolver { return &queryResolver{r} }

type mutationResolver struct{ *Resolver }
type queryResolver struct{ *Resolver }
