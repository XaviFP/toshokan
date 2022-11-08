package deck

func ValidateDecks(ds []Deck) (bool, []ErroredDeck) {
	erroredDecks := []ErroredDeck{}
	for _, d := range ds {
		if ok, ed := d.Validate(); !ok {
			erroredDecks = append(erroredDecks, ed)
		}
	}

	return len(erroredDecks) == 0, erroredDecks
}

func (d Deck) Validate() (bool, ErroredDeck) {
	ed := ErroredDeck{}
	ed.D = d
	if d.Title == "" {
		ed.Errs = append(ed.Errs, ErrNoTitle)
	}

	if d.Description == "" {
		ed.Errs = append(ed.Errs, ErrNoDescription)
	}

	var ok bool

	ok, ed.ErroredCards = ValidateCards(d.Cards)
	if !ok {
		ed.Errs = append(ed.Errs, ErrCards)
	}

	return len(ed.Errs) == 0, ed
}

func ValidateCards(cs []Card) (bool, []ErroredCard) {
	erroredCards := []ErroredCard{}
	for _, c := range cs {
		if ok, ec := ValidateCard(c); !ok {
			erroredCards = append(erroredCards, ec)
		}
	}

	return len(erroredCards) == 0, erroredCards
}

func ValidateCard(c Card) (bool, ErroredCard) {
	eq := ErroredCard{}
	eq.C = c
	if c.Title == "" {
		eq.Errs = append(eq.Errs, ErrNoTitle)
	}

	if len(c.PossibleAnswers) == 0 {
		eq.Errs = append(eq.Errs, ErrNoAnswersProvided)
	}

	var atLeastOneCorrectAnswer, noTextAnswer bool
	for _, a := range c.PossibleAnswers {
		if a.IsCorrect {
			atLeastOneCorrectAnswer = true
		}
		if a.Text == "" {
			noTextAnswer = true
		}
	}
	if !atLeastOneCorrectAnswer {
		eq.Errs = append(eq.Errs, ErrNoCorrectAnswer)
	}

	if noTextAnswer {
		eq.Errs = append(eq.Errs, ErrNoTextAnswer)
	}

	return len(eq.Errs) == 0, eq
}
