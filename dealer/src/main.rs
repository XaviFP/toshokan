use std::collections::HashMap;
use std::time::Duration;

use tokio::signal::unix::{signal, SignalKind};
use tokio_postgres::error::SqlState;
use tokio_postgres::NoTls;
use tokio_util::sync::CancellationToken;
use tonic::transport::{Channel, Endpoint, Server};
use tonic::{Code, Request, Response, Status};
use uuid::Uuid;

mod pb {
    include!("../api/proto/v1/v1.rs");
}

mod course_pb {
    include!("../api/proto/v1/course.v1.rs");
}

use course_pb::course_api_client::CourseApiClient;
use course_pb::{GetCourseDeckIdsRequest, GetLessonDeckIdsRequest};
use pb::dealer_server::{Dealer as pbDealer, DealerServer};
use pb::{
    DealRequest, DealResponse, ReviewAnswersRequest, ReviewAnswersResponse, StoreAnswersRequest,
    StoreAnswersResponse,
};

mod config;
use config::{load_course_service_config, load_db_config, load_grpc_server_config};

mod ranker;
use ranker::Ranker;

#[cfg(test)]
mod repository_it_test;

const LEVEL1_PERCENTATGE: usize = 40; //6
const LEVEL2_PERCENTATGE: usize = 20; //3
const LEVEL3_PERCENTATGE: usize = 15; //2
const LEVEL4_PERCENTATGE: usize = 15; //2
const LEVEL5_PERCENTATGE: usize = 10; //1

const COURSE_SERVICE_TIMEOUT: Duration = Duration::from_secs(5);

struct Dealer {
    repo: Repository,
    courses: Box<dyn CourseDecks>,
}

// Looks up which decks belong to a course or lesson, in the course service.
#[tonic::async_trait]
trait CourseDecks: Send + Sync + 'static {
    async fn course_deck_ids(&self, course_id: &Uuid) -> Result<Vec<Uuid>, Status>;
    async fn lesson_deck_ids(&self, lesson_id: &Uuid) -> Result<Vec<Uuid>, Status>;
}

#[tonic::async_trait]
impl CourseDecks for CourseApiClient<Channel> {
    async fn course_deck_ids(&self, course_id: &Uuid) -> Result<Vec<Uuid>, Status> {
        let response = self
            .clone()
            .get_course_deck_ids(GetCourseDeckIdsRequest {
                course_id: course_id.to_string(),
            })
            .await?;
        parse_deck_ids(&response.get_ref().deck_ids)
    }

    async fn lesson_deck_ids(&self, lesson_id: &Uuid) -> Result<Vec<Uuid>, Status> {
        let response = self
            .clone()
            .get_lesson_deck_ids(GetLessonDeckIdsRequest {
                lesson_id: lesson_id.to_string(),
            })
            .await?;
        parse_deck_ids(&response.get_ref().deck_ids)
    }
}

fn parse_deck_ids(deck_ids: &[String]) -> Result<Vec<Uuid>, Status> {
    deck_ids
        .iter()
        .map(|id| Uuid::try_parse(id).map_err(|_| Status::internal("invalid deck_id from course service")))
        .collect()
}
struct Repository {
    client: tokio_postgres::Client,
}

impl Repository {

    async fn get_max_cards_per_level(
        &self,
        user_id: &Uuid,
        deck_ids: &[Uuid],
        max: &i64,
    ) -> Result<Vec<LeveledCard>, tokio_postgres::Error> {
        let rows = self
            .client
            .query(
                "WITH user_cards as (
                    SELECT cards.id as card_id, user_card_level.lvl as lvl, random() as shuffle
                    FROM cards
                    JOIN decks
                    ON decks.id = cards.deck_id
                    AND decks.deleted_at IS NULL
                    JOIN user_card_level
                    ON user_card_level.card_id = cards.id
                    AND user_card_level.user_id = $1
                    WHERE (cardinality($2::uuid[]) = 0 OR cards.deck_id = ANY($2::uuid[]))
                    AND cards.deleted_at IS NULL
                )
                (SELECT card_id, lvl, shuffle FROM user_cards WHERE lvl = 1 ORDER BY shuffle LIMIT $3)
                UNION ALL (SELECT card_id, lvl, shuffle FROM user_cards WHERE lvl = 2 ORDER BY shuffle LIMIT $3)
                UNION ALL (SELECT card_id, lvl, shuffle FROM user_cards WHERE lvl = 3 ORDER BY shuffle LIMIT $3)
                UNION ALL (SELECT card_id, lvl, shuffle FROM user_cards WHERE lvl = 4 ORDER BY shuffle LIMIT $3)
                UNION ALL (SELECT card_id, lvl, shuffle FROM user_cards WHERE lvl = 5 ORDER BY shuffle LIMIT $3)
                ORDER BY lvl, shuffle",
                &[user_id, &deck_ids, max],
            )
            .await?;
        Ok(rows
            .iter()
            .map(|lvl_card| {
                let id: Uuid = lvl_card.get(0);
                let lvl: i32 = lvl_card.get(1);

                LeveledCard {
                    id: id.to_string(),
                    lvl: lvl as usize,
                }
            })
            .collect())
    }

    async fn store_answers(
        &self,
        user_id: &Uuid,
        answer_ids: &[Uuid],
    ) -> Result<(), tokio_postgres::Error> {
        // A card answered more than once in the same call counts as correct only if every answer was.
        self.client
            .execute(
                "WITH practice AS (
                    INSERT INTO card_practice (user_id, answer_id)
                    SELECT $1::uuid, unnest($2::uuid[])
                ),
                answered_cards AS (
                    SELECT card_id, bool_and(is_correct) AS is_correct
                    FROM answers
                    WHERE id = ANY($2::uuid[])
                    GROUP BY card_id
                )
                INSERT INTO user_card_level (user_id, card_id, lvl, updated_at)
                SELECT $1::uuid, a.card_id,
                    CASE WHEN a.is_correct
                        THEN LEAST(COALESCE(l.lvl, 1) + 1, 5)
                        ELSE GREATEST(COALESCE(l.lvl, 1) - 1, 1)
                    END,
                    now()
                FROM answered_cards a
                LEFT JOIN user_card_level l
                ON l.card_id = a.card_id
                AND l.user_id = $1::uuid
                ON CONFLICT ON CONSTRAINT user_card_level_card_id_user_id_key DO UPDATE
                SET lvl = EXCLUDED.lvl, updated_at = EXCLUDED.updated_at",
                &[user_id, &answer_ids],
            )
            .await?;

        Ok(())
    }

    // Unknown answers and answers without a card count as outside the pool too.
    async fn count_answers_outside_pool(
        &self,
        user_id: &Uuid,
        answer_ids: &[Uuid],
    ) -> Result<i64, tokio_postgres::Error> {
        let row = self
            .client
            .query_one(
                "SELECT count(*)
                FROM unnest($2::uuid[]) AS submitted(answer_id)
                LEFT JOIN answers a ON a.id = submitted.answer_id
                LEFT JOIN user_card_level l
                ON l.card_id = a.card_id
                AND l.user_id = $1::uuid
                WHERE l.card_id IS NULL",
                &[user_id, &answer_ids],
            )
            .await?;

        Ok(row.get(0))
    }
}

#[derive(Debug, PartialEq)]
enum Scope {
    All,
    Decks(Vec<Uuid>),
    Course(Uuid),
    Lesson(Uuid),
}

struct LeveledCard {
    id: String,
    lvl: usize,
}

fn parse_uuid(field: &str, value: &str) -> Result<Uuid, Status> {
    Uuid::try_parse(value).map_err(|_| Status::invalid_argument(format!("invalid {}", field)))
}

fn parse_answers_request(user_id: &str, answer_ids: &[String]) -> Result<(Uuid, Vec<Uuid>), Status> {
    let user_id = parse_uuid("user_id", user_id)?;
    if answer_ids.is_empty() {
        return Err(Status::invalid_argument("no answers provided"));
    }
    let answer_ids = answer_ids
        .iter()
        .map(|id| parse_uuid("answer_id", id))
        .collect::<Result<Vec<Uuid>, Status>>()?;

    Ok((user_id, answer_ids))
}

fn parse_deal_request(request: &DealRequest) -> Result<(Uuid, Scope), Status> {
    let user_id = parse_uuid("user_id", &request.user_id)?;
    let scopes_set = [
        !request.deck_ids.is_empty(),
        !request.course_id.is_empty(),
        !request.lesson_id.is_empty(),
    ]
    .iter()
    .filter(|set| **set)
    .count();
    if scopes_set > 1 {
        return Err(Status::invalid_argument(
            "set at most one of deck_ids, course_id and lesson_id",
        ));
    }

    if !request.course_id.is_empty() {
        return Ok((user_id, Scope::Course(parse_uuid("course_id", &request.course_id)?)));
    }
    if !request.lesson_id.is_empty() {
        return Ok((user_id, Scope::Lesson(parse_uuid("lesson_id", &request.lesson_id)?)));
    }
    if request.deck_ids.is_empty() {
        return Ok((user_id, Scope::All));
    }

    let deck_ids = request
        .deck_ids
        .iter()
        .map(|id| parse_uuid("deck_id", id))
        .collect::<Result<Vec<Uuid>, Status>>()?;

    Ok((user_id, Scope::Decks(deck_ids)))
}

fn internal_error(operation: &str, err: tokio_postgres::Error) -> Status {
    eprintln!("{} failed: {}", operation, err);
    Status::internal(format!("{} failed", operation))
}

// Not found and invalid argument are the caller's mistake, so they pass through as is.
fn course_service_error(err: Status) -> Status {
    match err.code() {
        Code::NotFound | Code::InvalidArgument => err,
        _ => {
            eprintln!("course service lookup failed: {}", err);
            Status::internal("course service lookup failed")
        }
    }
}

fn is_null_card_id(err: &tokio_postgres::Error) -> bool {
    err.code() == Some(&SqlState::NOT_NULL_VIOLATION)
        && err.as_db_error().and_then(|db_err| db_err.column()) == Some("card_id")
}

#[tonic::async_trait]
impl pbDealer for Dealer {
    async fn store_answers(
        &self,
        request: Request<StoreAnswersRequest>,
    ) -> Result<Response<StoreAnswersResponse>, Status> {
        let request = request.get_ref();
        let (user_id, answer_ids) = parse_answers_request(&request.user_id, &request.answer_ids)?;

        match self.repo.store_answers(&user_id, &answer_ids).await {
            Ok(()) => Ok(Response::new(StoreAnswersResponse {})),
            Err(err) if err.code() == Some(&SqlState::FOREIGN_KEY_VIOLATION) => {
                Err(Status::invalid_argument("unknown answer_id"))
            }
            Err(err) if is_null_card_id(&err) => {
                Err(Status::invalid_argument("answer_id does not belong to a card"))
            }
            Err(err) => Err(internal_error("store answers", err)),
        }
    }

    async fn review_answers(
        &self,
        request: Request<ReviewAnswersRequest>,
    ) -> Result<Response<ReviewAnswersResponse>, Status> {
        let request = request.get_ref();
        let (user_id, answer_ids) = parse_answers_request(&request.user_id, &request.answer_ids)?;

        // Cards never leave a pool, so the check can't go stale before the write.
        let outside_pool = self
            .repo
            .count_answers_outside_pool(&user_id, &answer_ids)
            .await
            .map_err(|err| internal_error("review answers", err))?;
        if outside_pool > 0 {
            return Err(Status::invalid_argument("answer_id not in review pool"));
        }

        self.repo
            .store_answers(&user_id, &answer_ids)
            .await
            .map_err(|err| internal_error("review answers", err))?;

        Ok(Response::new(ReviewAnswersResponse {}))
    }

    async fn deal(&self, request: Request<DealRequest>) -> Result<Response<DealResponse>, Status> {
        let (user_id, scope) = parse_deal_request(request.get_ref())?;
        let whole_pool = scope == Scope::All;
        let deck_ids = match scope {
            Scope::All => vec![],
            Scope::Decks(deck_ids) => deck_ids,
            Scope::Course(course_id) => self
                .courses
                .course_deck_ids(&course_id)
                .await
                .map_err(course_service_error)?,
            Scope::Lesson(lesson_id) => self
                .courses
                .lesson_deck_ids(&lesson_id)
                .await
                .map_err(course_service_error)?,
        };
        // A course or lesson without decks deals nothing; an empty list would mean every deck to the query.
        if deck_ids.is_empty() && !whole_pool {
            return Ok(Response::new(DealResponse { card_ids: vec![] }));
        }
        let max = i64::from(request.get_ref().number_of_cards);
        let level_cards = self
            .repo
            .get_max_cards_per_level(&user_id, &deck_ids, &max)
            .await
            .map_err(|err| internal_error("deal", err))?;
        if level_cards.len() <= request.get_ref().number_of_cards as usize {
            return Ok(Response::new(DealResponse {
                card_ids: level_cards.iter().map(|card| card.id.clone()).collect(),
            }));
        }

        let out = get_user_cards(level_cards, request.get_ref().number_of_cards as usize);
        return Ok(Response::new(DealResponse {
            card_ids: out.iter().map(|id| id.clone()).collect(),
        }));
    }
}

pub struct Deck {
    pub description: String,
    pub title: String,
}

#[tokio::main]
async fn main() -> Result<(), Box<dyn std::error::Error>> {
    // Load config
    let db_conf = load_db_config();
    if db_conf.is_err() {
        panic!("{}", db_conf.err().unwrap());
    }
    let db_config = db_conf.unwrap();

    let grpc_config = load_grpc_server_config();
    if grpc_config.is_err() {
        panic!("{}", grpc_config.err().unwrap());
    }

    // Stablish DB connection for ranker job
    let (ranker_client, ranker_connection) =
        tokio_postgres::connect(&db_config.to_string(), NoTls).await?;

    // Spawn connection for ranker job
    tokio::spawn(async move {
        if let Err(error) = ranker_connection.await {
            eprintln!("Connection error: {}", error);
        }
    });

    let ranker = Ranker::new(ranker_client);

    // Create cancellation token for graceful shutdown
    let cancel_token = CancellationToken::new();
    let ranker_token = cancel_token.clone();

    tokio::spawn(async move {
        if let Err(error) = ranker.start(ranker_token).await {
            eprintln!("Ranker error: {}", error);
        }
    });

    // Stablish DB connection
    let (client, connection) = tokio_postgres::connect(&db_config.to_string(), NoTls).await?;

    // Spawn connection
    tokio::spawn(async move {
        if let Err(error) = connection.await {
            eprintln!("Connection error: {}", error);
        }
    });

    // Start GRPC Server
    let addr = grpc_config.unwrap().to_string().parse()?;
    println!("Starting Dealer gRPC server on: {}", addr);

    let course_config = load_course_service_config();
    if course_config.is_err() {
        panic!("{}", course_config.err().unwrap());
    }
    // Lazy, so dealer starts even when the course service isn't up yet.
    let course_channel = Endpoint::from_shared(course_config.unwrap().to_string())?
        .timeout(COURSE_SERVICE_TIMEOUT)
        .connect_lazy();

    let dealer = Dealer {
        repo: Repository { client },
        courses: Box::new(CourseApiClient::new(course_channel)),
    };

    Server::builder()
        .add_service(DealerServer::new(dealer))
        .serve_with_shutdown(addr, async move {
            let mut sigterm = signal(SignalKind::terminate()).expect("Failed to register SIGTERM handler");
            let mut sigint = signal(SignalKind::interrupt()).expect("Failed to register SIGINT handler");
            tokio::select! {
                _ = sigterm.recv() => println!("Received SIGTERM, stopping server..."),
                _ = sigint.recv() => println!("Received SIGINT, stopping server..."),
            }
            cancel_token.cancel();
        })
        .await?;

    Ok(())
}

fn get_user_cards(cards: Vec<LeveledCard>, max: usize) -> Vec<String> {
    let mut level_map: HashMap<usize, Vec<&String>> = HashMap::new();

    // Initialize map with all levels
    for lvl in 1..=5 {
        level_map.insert(lvl, vec![]);
    }

    for card in cards.iter() {
        if let Some(entry) = level_map.get_mut(&card.lvl) {
            entry.push(&card.id);
        }
    }

    let mut out: Vec<String> = vec![];
    let mut missing: usize = 0;
    for lvl in 1..=5 {
        let lvl_count = get_count_for_level(max, lvl);
        let entry = level_map.get_mut(&lvl);
        let ids = entry.unwrap();
        let mut length = ids.len();
        if length == 0 {
            continue;
        }
        if length < lvl_count {
            missing = missing + (lvl_count - length);
            ids.drain(..length).for_each(|id| out.push(id.into()));

            continue;
        }
        // Maybe randomize?
        ids.drain(..lvl_count).for_each(|id| out.push(id.into()));

        if missing > 0 {
            // There are extra slots form previous levels and extra cards to fill them
            length = ids.len();
            match missing >= length {
                // There are at most the same cards as slots
                true => {
                    missing = missing - length;
                    ids.drain(..).for_each(|id| out.push(id.into()));
                }
                // There are more cards than slots
                false => {
                    ids.drain(..missing).for_each(|id| out.push(id.into()));
                    missing = 0;
                }
            }
        }
    }
    // All cards requested are there
    let length = out.len();
    if max == length {
        return out;
    }
    // Remove extra cards if any
    if max < length {
        let exceeding = length - max;
        for _ in 0..exceeding {
            out.pop();
        }
        return out;
    }
    // Fill the rest of slots in reverse level order
    missing = max - out.len();
    for lvl in (1..=5).rev() {
        let entry = level_map.get_mut(&lvl);
        let ids = entry.unwrap();
        let len = ids.len();

        if len == 0 {
            continue;
        }
        if missing > len {
            missing = missing - len;
            ids.drain(..).for_each(|id| out.push(id.into()));

            continue;
        }

        ids.drain(0..missing).for_each(|id| out.push(id.into()));

        break;
    }
    out
}

fn get_count_for_level(max: usize, lvl: usize) -> usize {
    match lvl {
        1 => max * LEVEL1_PERCENTATGE / 100,
        2 => max * LEVEL2_PERCENTATGE / 100,
        3 => max * LEVEL3_PERCENTATGE / 100,
        4 => max * LEVEL4_PERCENTATGE / 100,
        5 => max * LEVEL5_PERCENTATGE / 100,
        _ => 0,
    }
}

fn _create_test_level_cards(max: usize) -> (Vec<LeveledCard>, HashMap<String, usize>) {
    let mut out: Vec<LeveledCard> = vec![];
    let mut level_map: HashMap<String, usize> = HashMap::new();
    for i in 0..max {
        out.push(LeveledCard {
            id: i.to_string(),
            lvl: (i % 5) + 1,
        });
        level_map.insert(i.to_string(), (i % 5) + 1);
    }
    (out, level_map)
}

#[test]
fn number_of_cards_returned_matches_input() {
    let max: usize = 15;
    let (cards, _) = _create_test_level_cards(max * 3);
    assert_eq!(cards.len(), 3 * max);
    let out = get_user_cards(cards, max);
    assert_eq!(out.len(), max);
}

#[test]
fn correct_number_of_cards_per_category() {
    let max: usize = 15;
    let (cards, level_map) = _create_test_level_cards(max * 3);
    assert_eq!(cards.len(), 3 * max);
    let out = get_user_cards(cards, max);

    let mut count_per_level: HashMap<usize, usize> = HashMap::new();
    for id in out.iter() {
        let count = count_per_level
            .entry(level_map.get(id).unwrap().clone())
            .or_insert(0);
        *count += 1;
    }
    for level in 1..=5 {
        assert_eq!(
            get_count_for_level(max, level) <= *count_per_level.get(&level).unwrap(),
            true
        )
    }
    assert_eq!(out.len().clone(), max);
}

#[test]
fn all_cards_are_returned_if_less_than_requested() {
    let max: usize = 20;
    let actual: usize = 15;
    let (cards, _) = _create_test_level_cards(actual);
    let out = get_user_cards(cards, max);
    assert_eq!(out.len(), actual)
}

#[cfg(test)]
mod validation_test {
    use tonic::Code;
    use uuid::{uuid, Uuid};

    use super::pb::DealRequest;
    use super::{parse_answers_request, parse_deal_request, Scope};

    const USER: Uuid = uuid!("11111111-0000-0000-0000-000000000000");
    const DECK: Uuid = uuid!("dddddddd-0000-0000-0000-000000000000");
    const OTHER_DECK: Uuid = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    const COURSE: Uuid = uuid!("c0c0c0c0-0000-0000-0000-000000000000");
    const LESSON: Uuid = uuid!("1e1e1e1e-0000-0000-0000-000000000000");
    const ANSWER_A_CORRECT: Uuid = uuid!("aaaaaaaa-0000-0000-0000-00000000000c");
    const ANSWER_B_CORRECT: Uuid = uuid!("bbbbbbbb-0000-0000-0000-00000000000c");

    #[test]
    fn parse_deal_request_rejects_invalid_user_id() {
        let actual = parse_deal_request(&DealRequest {
            user_id: "not-a-uuid".to_owned(),
            deck_ids: vec![DECK.to_string()],
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_deal_request_rejects_invalid_deck_id() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            deck_ids: vec![DECK.to_string(), "not-a-uuid".to_owned()],
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_deal_request_rejects_invalid_course_id() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            course_id: "not-a-uuid".to_owned(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_deal_request_rejects_invalid_lesson_id() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            lesson_id: "not-a-uuid".to_owned(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_deal_request_rejects_decks_and_course_together() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            deck_ids: vec![DECK.to_string()],
            course_id: COURSE.to_string(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_deal_request_rejects_course_and_lesson_together() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            course_id: COURSE.to_string(),
            lesson_id: LESSON.to_string(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_deal_request_accepts_decks() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            deck_ids: vec![DECK.to_string(), OTHER_DECK.to_string()],
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap();

        assert_eq!((USER, Scope::Decks(vec![DECK, OTHER_DECK])), actual);
    }

    #[test]
    fn parse_deal_request_accepts_course() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            course_id: COURSE.to_string(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap();

        assert_eq!((USER, Scope::Course(COURSE)), actual);
    }

    #[test]
    fn parse_deal_request_accepts_lesson() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            lesson_id: LESSON.to_string(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap();

        assert_eq!((USER, Scope::Lesson(LESSON)), actual);
    }

    #[test]
    fn parse_deal_request_accepts_no_scope() {
        let actual = parse_deal_request(&DealRequest {
            user_id: USER.to_string(),
            number_of_cards: 5,
            ..Default::default()
        })
        .unwrap();

        assert_eq!((USER, Scope::All), actual);
    }

    #[test]
    fn parse_answers_request_rejects_invalid_user_id() {
        let actual =
            parse_answers_request("not-a-uuid", &[ANSWER_A_CORRECT.to_string()]).unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_answers_request_rejects_invalid_answer_id() {
        let actual = parse_answers_request(
            &USER.to_string(),
            &[ANSWER_A_CORRECT.to_string(), "not-a-uuid".to_owned()],
        )
        .unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_answers_request_rejects_empty_answers() {
        let actual = parse_answers_request(&USER.to_string(), &[]).unwrap_err();

        assert_eq!(Code::InvalidArgument, actual.code());
    }

    #[test]
    fn parse_answers_request_accepts_valid_ids() {
        let actual = parse_answers_request(
            &USER.to_string(),
            &[ANSWER_A_CORRECT.to_string(), ANSWER_B_CORRECT.to_string()],
        )
        .unwrap();

        assert_eq!((USER, vec![ANSWER_A_CORRECT, ANSWER_B_CORRECT]), actual);
    }
}
