package grpc

import (
	"context"
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
		return &pb.GetDeckResponse{}, errors.Trace(err)
	}

	d, err := s.Repository.GetDeck(ctx, deckID)
	if err != nil {
		if errors.Cause(err) == deck.ErrDeckNotFound {

			return &pb.GetDeckResponse{}, status.Error(codes.NotFound, errors.Trace(err).Error())
		}
		return &pb.GetDeckResponse{}, errors.Trace(err)
	}

	if (d.AuthorID.String() != req.UserId) && !d.Public {
		return nil, errors.New("not authorized")
	}

	return &pb.GetDeckResponse{Deck: toGRPCDeck(d)}, nil
}

func (s *Server) GetDecks(ctx context.Context, req *pb.GetDecksRequest) (*pb.GetDecksResponse, error) {
	var ids []uuid.UUID

	for _, idStr := range req.DeckIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, errors.Trace(err)
		}

		ids = append(ids, id)
	}

	decks, err := s.Repository.GetDecks(ctx, ids)
	if err != nil {
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
		return nil, errors.Trace(err)
	}

	res, err := s.Repository.GetPopularDecks(ctx, userID, paginationFromProto(req.Pagination))
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &pb.GetPopularDecksResponse{
		Connection: connectionToProto(res),
	}, nil
}

func (s *Server) CreateCard(ctx context.Context, req *pb.CreateCardRequest) (*pb.CreateCardResponse, error) {
	deckID, err := uuid.Parse(req.Card.DeckId)
	if err != nil {
		return nil, errors.Trace(err)
	}

	card, err := fromGRPCCard(req.Card)
	if err != nil {
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
			return &pb.CreateCardResponse{}, status.Error(codes.NotFound, errors.Trace(err).Error())
		}
		return &pb.CreateCardResponse{}, errors.Trace(err)
	}

	card.GenerateUUIDs()

	err = s.Repository.StoreCard(ctx, card, deckID)
	if err != nil {
		return nil, errors.Trace(err)
	}

	return &pb.CreateCardResponse{Card: toGRPCCard(card)}, nil
}

func (s *Server) GetCards(ctx context.Context, req *pb.GetCardsRequest) (*pb.GetCardsResponse, error) {
	var ids []uuid.UUID

	for _, idStr := range req.CardIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, errors.Trace(err)
		}

		ids = append(ids, id)
	}

	cards, err := s.Repository.GetCards(ctx, ids)
	if err != nil {
		return &pb.GetCardsResponse{}, errors.Trace(err)
	}

	out := make(map[string]*pb.Card, len(cards))

	for id, c := range cards {
		out[id.String()] = &pb.Card{
			Id:              c.ID.String(),
			Title:           c.Title,
			Explanation:     c.Explanation,
			PossibleAnswers: toGRPCAnswers(c.PossibleAnswers),
		}
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
		return nil, errors.Trace(err)
	}

	isValid, _ := d.Validate()
	if !isValid {
		return &pb.CreateDeckResponse{}, deck.ErrDeckInvalid
	}

	d.GenerateUUIDs()

	if err := s.Repository.StoreDeck(ctx, d); err != nil {
		return &pb.CreateDeckResponse{}, errors.Trace(err)
	}

	return &pb.CreateDeckResponse{Deck: toGRPCDeck(d)}, nil
}

func (s *Server) DeleteDeck(ctx context.Context, req *pb.DeleteDeckRequest) (*pb.DeleteDeckResponse, error) {
	deckID, err := uuid.Parse(req.Id)
	if err != nil {
		return &pb.DeleteDeckResponse{}, errors.Trace(err)
	}

	err = s.Repository.DeleteDeck(ctx, deckID)

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
			return deck.Deck{}, errors.Trace(err)
		}

	}

	authorID, err := uuid.Parse(d.AuthorId)
	if err != nil {
		return deck.Deck{}, errors.Trace(err)
	}

	cards, err := fromGRPCCards(d.Cards)
	if err != nil {
		return deck.Deck{}, errors.Trace(err)
	}

	return deck.Deck{
		ID:          deckID,
		AuthorID:    authorID,
		Title:       d.Title,
		Description: d.Description,
		Cards:       cards,
	}, nil
}

func fromGRPCCards(cards []*pb.Card) ([]deck.Card, error) {
	var out = make([]deck.Card, 0, len(cards))

	for _, c := range cards {
		converted, err := fromGRPCCard(c)
		if err != nil {
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
			return deck.Card{}, errors.Trace(err)
		}
	}

	answers, err := fromGRPCAnswers(c.PossibleAnswers)
	if err != nil {
		return deck.Card{}, errors.Trace(err)
	}

	return deck.Card{
		ID:              cardID,
		Title:           c.Title,
		PossibleAnswers: answers,
		Explanation:     c.Explanation,
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
