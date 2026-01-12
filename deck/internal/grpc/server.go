package grpc

import (
	"context"
	"log/slog"
	"net"

	"github.com/google/uuid"
	"github.com/juju/errors"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/XaviFP/toshokan/common/pagination"
	pb "github.com/XaviFP/toshokan/deck/api/proto/v1"
	"github.com/XaviFP/toshokan/deck/internal/deck"
)

type Server struct {
	GRPCAddr      string
	GRPCTransport string

	Repository deck.Repository

	grpcServer *grpc.Server
}

func (s *Server) Start() error {
	s.grpcServer = grpc.NewServer()
	pb.RegisterDecksAPIServer(s.grpcServer, s)

	listener, err := net.Listen(s.GRPCTransport, s.GRPCAddr)
	if err != nil {
		return errors.Annotatef(err, "failed to listen on %s", listener.Addr())
	}
	defer listener.Close()

	if err := s.grpcServer.Serve(listener); err != nil {
		return errors.Annotatef(err, "failed to serve on %s", listener.Addr())
	}

	return nil
}

func (s *Server) Stop() {
	s.grpcServer.GracefulStop()
}

func (s *Server) GetDeck(ctx context.Context, req *pb.GetDeckRequest) (*pb.GetDeckResponse, error) {
	deckID, err := uuid.Parse(req.DeckId)
	if err != nil {
		slog.Error("GetDeck: failed to parse deck ID", "error", err, "deckId", req.DeckId, "stack", errors.ErrorStack(err))
		return &pb.GetDeckResponse{}, errors.Trace(err)
	}

	d, err := s.Repository.GetDeck(ctx, deckID)
	if err != nil {
		if errors.Cause(err) == deck.ErrDeckNotFound {
			slog.Error("GetDeck: deck not found", "error", err, "deckId", deckID.String())
			return &pb.GetDeckResponse{}, status.Error(codes.NotFound, errors.Trace(err).Error())
		}
		slog.Error("GetDeck: failed to get deck from repository", "error", err, "deckId", deckID.String(), "stack", errors.ErrorStack(err))
		return &pb.GetDeckResponse{}, errors.Trace(err)
	}

	// TODO: Provide an "internal" GetDeck that doesn't require authorization/ownership check
	// For services that need to access decks regardless of public/private status
	if (d.AuthorID.String() != req.UserId) && !d.Public {
		slog.Error("GetDeck: private deck access denied", "deckId", deckID.String(), "authorId", d.AuthorID.String(), "requestUserId", req.UserId)
		return nil, errors.New("private deck access denied")
	}

	return &pb.GetDeckResponse{Deck: toGRPCDeck(d)}, nil
}

func (s *Server) GetDecks(ctx context.Context, req *pb.GetDecksRequest) (*pb.GetDecksResponse, error) {
	var ids []uuid.UUID

	for _, idStr := range req.DeckIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			slog.Error("GetDecks: failed to parse deck ID", "error", err, "deckId", idStr, "stack", errors.ErrorStack(err))
			return nil, errors.Trace(err)
		}

		ids = append(ids, id)
	}

	decks, err := s.Repository.GetDecks(ctx, ids)
	if err != nil {
		slog.Error("GetDecks: failed to get decks from repository", "error", err, "deckCount", len(ids), "stack", errors.ErrorStack(err))
		return &pb.GetDecksResponse{}, errors.Trace(err)
	}

	out := make(map[string]*pb.Deck, len(decks))

	for id, deck := range decks {
		out[id.String()] = toGRPCDeck(deck)
	}

	return &pb.GetDecksResponse{Decks: out}, nil
}

func (s *Server) GetPopularDecks(ctx context.Context, req *pb.GetPopularDecksRequest) (*pb.GetPopularDecksResponse, error) {
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		slog.Error("GetPopularDecks: failed to parse user ID", "error", err, "userId", req.UserId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	res, err := s.Repository.GetPopularDecks(ctx, userID, paginationFromProto(req.Pagination))
	if err != nil {
		slog.Error("GetPopularDecks: failed to get popular decks", "error", err, "userId", userID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.GetPopularDecksResponse{
		Connection: connectionToProto(res),
	}, nil
}

func (s *Server) CreateCard(ctx context.Context, req *pb.CreateCardRequest) (*pb.CreateCardResponse, error) {
	deckID, err := uuid.Parse(req.Card.DeckId)
	if err != nil {
		slog.Error("CreateCard: failed to parse deck ID", "error", err, "deckId", req.Card.DeckId, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	card, err := fromGRPCCard(req.Card)
	if err != nil {
		slog.Error("CreateCard: failed to convert card from gRPC", "error", err, "deckId", deckID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	isValid, _ := deck.ValidateCard(card)
	if !isValid {
		return &pb.CreateCardResponse{}, deck.ErrCardInvalid
	}

	// Check if deck exists
	_, err = s.Repository.GetDeck(ctx, deckID)
	if err != nil {
		if errors.Cause(err) == deck.ErrDeckNotFound {
			slog.Error("CreateCard: deck not found", "error", err, "deckId", deckID.String())
			return &pb.CreateCardResponse{}, status.Error(codes.NotFound, errors.Trace(err).Error())
		}
		slog.Error("CreateCard: failed to get deck", "error", err, "deckId", deckID.String(), "stack", errors.ErrorStack(err))
		return &pb.CreateCardResponse{}, errors.Trace(err)
	}

	card.GenerateUUIDs()

	err = s.Repository.StoreCard(ctx, card, deckID)
	if err != nil {
		slog.Error("CreateCard: failed to store card", "error", err, "deckId", deckID.String(), "cardId", card.ID.String(), "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	return &pb.CreateCardResponse{Card: toGRPCCard(card)}, nil
}

func (s *Server) GetCards(ctx context.Context, req *pb.GetCardsRequest) (*pb.GetCardsResponse, error) {
	var ids []uuid.UUID

	for _, idStr := range req.CardIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			slog.Error("GetCards: failed to parse card ID", "error", err, "cardId", idStr, "stack", errors.ErrorStack(err))
			return nil, errors.Trace(err)
		}

		ids = append(ids, id)
	}

	cards, err := s.Repository.GetCards(ctx, ids)
	if err != nil {
		slog.Error("GetCards: failed to get cards from repository", "error", err, "cardCount", len(ids), "stack", errors.ErrorStack(err))
		return &pb.GetCardsResponse{}, errors.Trace(err)
	}

	out := make(map[string]*pb.Card, len(cards))

	for id, c := range cards {
		out[id.String()] = toGRPCCard(c)
	}

	return &pb.GetCardsResponse{Cards: out}, nil
}

func paginationFromProto(p *pb.Pagination) pagination.Pagination {
	return pagination.Pagination{
		Before: pagination.Cursor(p.Before),
		After:  pagination.Cursor(p.After),
		First:  int(p.First),
		Last:   int(p.Last),
	}
}

func connectionToProto(conn deck.PopularDecksConnection) *pb.PopularDecksConnection {
	var edges []*pb.PopularDecksConnection_Edge

	for _, e := range conn.Edges {
		edges = append(edges, &pb.PopularDecksConnection_Edge{
			DeckId: e.DeckID.String(),
			Cursor: string(e.Cursor),
		})
	}

	return &pb.PopularDecksConnection{
		Edges: edges,
		PageInfo: &pb.PageInfo{
			HasPreviousPage: conn.PageInfo.HasPreviousPage,
			HasNextPage:     conn.PageInfo.HasNextPage,
			StartCursor:     string(conn.PageInfo.StartCursor),
			EndCursor:       string(conn.PageInfo.EndCursor),
		},
	}
}

func (s *Server) CreateDeck(ctx context.Context, req *pb.CreateDeckRequest) (*pb.CreateDeckResponse, error) {
	d, err := fromGRPCDeck(req.Deck)
	if err != nil {
		slog.Error("CreateDeck: failed to convert deck from gRPC", "error", err, "stack", errors.ErrorStack(err))
		return nil, errors.Trace(err)
	}

	isValid, _ := d.Validate()
	if !isValid {
		return &pb.CreateDeckResponse{}, deck.ErrDeckInvalid
	}

	d.GenerateUUIDs()

	if err := s.Repository.StoreDeck(ctx, d); err != nil {
		slog.Error("CreateDeck: failed to store deck", "error", err, "deckId", d.ID.String(), "stack", errors.ErrorStack(err))
		return &pb.CreateDeckResponse{}, errors.Trace(err)
	}

	return &pb.CreateDeckResponse{Deck: toGRPCDeck(d)}, nil
}

func (s *Server) DeleteDeck(ctx context.Context, req *pb.DeleteDeckRequest) (*pb.DeleteDeckResponse, error) {
	deckID, err := uuid.Parse(req.Id)
	if err != nil {
		slog.Error("DeleteDeck: failed to parse deck ID", "error", err, "deckId", req.Id, "stack", errors.ErrorStack(err))
		return &pb.DeleteDeckResponse{}, errors.Trace(err)
	}

	err = s.Repository.DeleteDeck(ctx, deckID)
	if err != nil {
		slog.Error("DeleteDeck: failed to delete deck", "error", err, "deckId", deckID.String(), "stack", errors.ErrorStack(err))
	}

	return &pb.DeleteDeckResponse{}, errors.Trace(err)
}

func toGRPCDecks(decks []deck.Deck) []*pb.Deck {
	var out = make([]*pb.Deck, 0, len(decks))

	for _, deck := range decks {
		out = append(out, toGRPCDeck(deck))
	}

	return out
}

func toGRPCDeck(d deck.Deck) *pb.Deck {
	return &pb.Deck{
		Id:          d.ID.String(),
		AuthorId:    d.AuthorID.String(),
		Title:       d.Title,
		Description: d.Description,
		IsPublic:    d.Public,
		Cards:       toGRPCCards(d.Cards),
	}
}

func toGRPCCards(cards []deck.Card) []*pb.Card {
	var out = make([]*pb.Card, 0, len(cards))

	for _, c := range cards {
		out = append(out, toGRPCCard(c))
	}

	return out
}

func toGRPCCard(c deck.Card) *pb.Card {
	return &pb.Card{
		Id:              c.ID.String(),
		Title:           c.Title,
		PossibleAnswers: toGRPCAnswers(c.PossibleAnswers),
		Explanation:     c.Explanation,
		Kind:            c.Kind,
	}
}

func toGRPCAnswers(answers []deck.Answer) []*pb.Answer {
	var out = make([]*pb.Answer, 0, len(answers))

	for _, a := range answers {
		out = append(out, &pb.Answer{
			Id:        a.ID.String(),
			Text:      a.Text,
			IsCorrect: a.IsCorrect,
		})
	}

	return out
}

func fromGRPCDeck(d *pb.Deck) (deck.Deck, error) {
	if d == nil {
		return deck.Deck{}, nil
	}

	var deckID uuid.UUID

	if d.Id != "" {
		var err error

		deckID, err = uuid.Parse(d.Id)
		if err != nil {
			slog.Error("fromGRPCDeck: failed to parse deck ID", "error", err, "deckId", d.Id, "stack", errors.ErrorStack(err))
			return deck.Deck{}, errors.Trace(err)
		}

	}

	authorID, err := uuid.Parse(d.AuthorId)
	if err != nil {
		slog.Error("fromGRPCDeck: failed to parse author ID", "error", err, "authorId", d.AuthorId, "stack", errors.ErrorStack(err))
		return deck.Deck{}, errors.Trace(err)
	}

	cards, err := fromGRPCCards(d.Cards)
	if err != nil {
		slog.Error("fromGRPCDeck: failed to convert cards", "error", err, "cardCount", len(d.Cards), "stack", errors.ErrorStack(err))
		return deck.Deck{}, errors.Trace(err)
	}

	return deck.Deck{
		ID:          deckID,
		AuthorID:    authorID,
		Title:       d.Title,
		Description: d.Description,
		Cards:       cards,
		Public:      d.IsPublic,
	}, nil
}

func fromGRPCCards(cards []*pb.Card) ([]deck.Card, error) {
	var out = make([]deck.Card, 0, len(cards))

	for _, c := range cards {
		converted, err := fromGRPCCard(c)
		if err != nil {
			slog.Error("fromGRPCCards: failed to convert card", "error", err, "cardTitle", c.Title, "stack", errors.ErrorStack(err))
			return []deck.Card{}, errors.Trace(err)
		}

		out = append(out, converted)
	}

	return out, nil
}

func fromGRPCCard(c *pb.Card) (deck.Card, error) {
	var cardID uuid.UUID

	if c.Id != "" {
		var err error

		cardID, err = uuid.Parse(c.Id)
		if err != nil {
			slog.Error("fromGRPCCard: failed to parse card ID", "error", err, "cardId", c.Id, "stack", errors.ErrorStack(err))
			return deck.Card{}, errors.Trace(err)
		}
	}

	answers, err := fromGRPCAnswers(c.PossibleAnswers)
	if err != nil {
		slog.Error("fromGRPCCard: failed to convert answers", "error", err, "answerCount", len(c.PossibleAnswers), "stack", errors.ErrorStack(err))
		return deck.Card{}, errors.Trace(err)
	}

	return deck.Card{
		ID:              cardID,
		Title:           c.Title,
		PossibleAnswers: answers,
		Explanation:     c.Explanation,
		Kind:            c.Kind,
	}, nil
}

func fromGRPCAnswers(answers []*pb.Answer) ([]deck.Answer, error) {
	var out = make([]deck.Answer, 0, len(answers))

	for _, a := range answers {
		var answerID uuid.UUID
		if a.Id != "" {
			var err error

			answerID, err = uuid.Parse(a.Id)
			if err != nil {
				slog.Error("fromGRPCAnswers: failed to parse answer ID", "error", err, "answerId", a.Id, "stack", errors.ErrorStack(err))
				return []deck.Answer{}, errors.Trace(err)
			}
		}

		out = append(out, deck.Answer{
			ID:        answerID,
			Text:      a.Text,
			IsCorrect: a.IsCorrect,
		})
	}

	return out, nil
}
