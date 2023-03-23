use std::collections::HashMap;
use std::fmt::Write;

use postgres::NoTls;
use postgres::types::ToSql;
use tonic::Code;
use tonic::{transport::Server, Request, Response, Status};
use uuid::{Uuid};

mod pb {
    include!("../api/proto/v1/v1.rs");
}

use pb::dealer_server::{Dealer as pbDealer, DealerServer};
use pb::{DealRequest, DealResponse, StoreAnswersRequest, StoreAnswersResponse};

mod config;
use config::{load_db_config, load_grpc_server_config};

const LEVEL1_PERCENTATGE: usize = 40; //6
const LEVEL2_PERCENTATGE: usize = 20; //3
const LEVEL3_PERCENTATGE: usize = 15; //2
const LEVEL4_PERCENTATGE: usize = 15; //2
const LEVEL5_PERCENTATGE: usize = 10; //1

struct Dealer {
    repo: Repository,
}
struct Repository {
    client: tokio_postgres::Client,
}

impl Repository {
    async fn is_new_deck_for_user(&self, user_id: &String, deck_id: &String) -> bool {

        let rows = self
            .client
            .query(
                "SELECT count(*) FROM user_deck WHERE user_id = $1 AND deck_id = $2",
                &[&Uuid::parse_str(user_id).unwrap(), &Uuid::parse_str(deck_id).unwrap()],
            )
            .await
            .unwrap_or(vec![]);

        let count: i64 = rows[0].get(0);

        return count == 0;
    }

    async fn init_levels(
        &self,
        user_id: &String,
        deck_id: &String,
        max: &i64,
    ) -> Result<Vec<String>, String> {

        let result = self
            .client
            .query(
                &format!("WITH inserted_cards as (INSERT INTO user_card_level (user_id, card_id)
                    (
                        SELECT '{}' as user_id, c.id as card_id
                        FROM cards as c
                        WHERE c.deck_id = $2
                    )
                    RETURNING card_id),
                    inserted_deck as (INSERT INTO user_deck (user_id, deck_id) VALUES ($1, $2))
                    SELECT card_id FROM inserted_cards LIMIT $3", user_id),
                &[&Uuid::parse_str(user_id).unwrap(), &Uuid::parse_str(deck_id).unwrap(), max],
            )
            .await;

        if result.as_ref().is_err() {
            match result.as_ref().err() {
                Some(err) => {
                    return Err(err.to_string());
                }
                _ => {}
            }
        }
        // If new deck for user, return request.number_of_cards first cards
        let card_uuids = result.as_ref().unwrap();
        let card_ids: Vec<Card> = card_uuids
            .iter()
            .map(|uuid| {
                let id: Uuid = uuid.get(0);
                Card { id: id.to_string() }
            })
            .collect();
        return Ok(card_ids.iter().map(|card| card.id.clone()).collect());
    }

    async fn get_max_cards_per_level(
        &self,
        user_id: &String,
        deck_id: &String,
        max: &i64,
    ) -> Result<Vec<LeveledCard>, String> {
        let rows = self
            .client
            .query(
                "WITH user_cards as (
                    SELECT c.id as card_id, uc.level as lvl
                    FROM cards as c 
                    JOIN user_card_level as uc 
                    ON uc.card_id = c.id
                    WHERE uc.user_id = $1
                    AND c.deck_id = $2
                    
                )
                (SELECT card_id, lvl FROM user_cards WHERE lvl = 1 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 2 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 3 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 4 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 5 LIMIT $3)
                ORDER BY lvl DESC",
                &[&Uuid::parse_str(user_id).unwrap(), &Uuid::parse_str(deck_id).unwrap(), max],
            )
            .await;
        if rows.as_ref().is_err() {
            match rows.as_ref().err() {
                Some(err) => {
                    return Err(err.to_string());
                }
                _ => {}
            }
        }
        Ok(rows
            .unwrap()
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

    async fn store_answers(&self, user_id: &String, answer_ids: &Vec<String>) -> Result<(), String> {
        let mut buf = String::from("INSERT INTO card_practice (user_id, answer_id) VALUES ");
        let mut i = 1;
        let mut params = Vec::<&(dyn ToSql + Sync)>::new();

        let uid_res = Uuid::try_parse(user_id);
        if uid_res.is_err() {
            return Err(uid_res.err().unwrap().to_string());
        }

        let uid = uid_res.unwrap();

        for answer in answer_ids.iter() {
            let res = buf.write_fmt(format_args!("(${},${})", i, i+1));
            if res.is_err() {
                return Err(res.err().unwrap().to_string());
            }
            params.push(&uid);
            params.push(answer);
            i += 2;
        }

        let result = self.client.execute(&buf, &params).await;
        if result.is_err() {
            return Err(result.err().unwrap().to_string());
        }

        // Update actual user_card_levels
        Ok(())
    }
}

struct Card {
    id: String,
}

struct LeveledCard {
    id: String,
    lvl: usize,
}

#[tonic::async_trait]
impl pbDealer for Dealer {
    async fn store_answers(
        &self,
        request: Request<StoreAnswersRequest>,
    ) -> Result<Response<StoreAnswersResponse>, Status> {
        let result = self.repo.store_answers(&request.get_ref().user_id, &request.get_ref().answers).await;
        if result.as_ref().is_err() {
            match result.as_ref().err() {
                Some(err) => {
                    return Err(Status::new(Code::Internal, format!("{}", err)));
                }
                _ => {}
            }
        }
        return Ok(Response::new(StoreAnswersResponse {}));
    }

    async fn deal(&self, request: Request<DealRequest>) -> Result<Response<DealResponse>, Status> {
        // If new deck for user, create user_card_level s for all cards
        if self
            .repo
            .is_new_deck_for_user(&request.get_ref().user_id, &request.get_ref().deck_id)
            .await
        {
            let max = i64::from(request.get_ref().number_of_cards);
            let result = self
                .repo
                .init_levels(
                    &request.get_ref().user_id,
                    &request.get_ref().deck_id.clone(),
                    &max,
                )
                .await;
            if result.as_ref().is_err() {
                match result.as_ref().err() {
                    Some(err) => {
                        return Err(Status::new(Code::Internal, format!("{}", err)));
                    }
                    _ => {}
                }
            }
            return Ok(Response::new(DealResponse {
                card_ids: result.unwrap().iter().map(|id| id.clone()).collect(),
            }));
        }
        // Apply algorithm to database result and return
        let max = i64::from(request.get_ref().number_of_cards);
        let result = self
            .repo
            .get_max_cards_per_level(
                &request.get_ref().user_id,
                &request.get_ref().deck_id,
                &max,
            )
            .await;
        if result.as_ref().is_err() {
            match result.as_ref().err() {
                Some(err) => {
                    return Err(Status::new(Code::Internal, format!("{}", err)));
                }
                _ => {}
            }
        }
        let level_cards = result.unwrap();
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
    let db_config = load_db_config();
    if db_config.is_err() {
        panic!("{}",db_config.err().unwrap());
    }

    let grpc_config = load_grpc_server_config();
    if grpc_config.is_err() {
        panic!("{}",grpc_config.err().unwrap());
    }

    // Stablish DB connection
    let (client, connection) = tokio_postgres::connect(
        &db_config.unwrap().to_string(),
        NoTls,
    )
    .await?;

    // Spawn connection
    tokio::spawn(async move {
        if let Err(error) = connection.await {
            eprintln!("Connection error: {}", error);
        }
    });

    // Start GRPC Server
    let addr = grpc_config.unwrap().to_string().parse()?;
    let dealer = Dealer {
        repo: Repository { client },
    };

    Server::builder()
        .add_service(DealerServer::new(dealer))
        .serve(addr)
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
        let count = count_per_level.entry(level_map.get(id).unwrap().clone()).or_insert(0);
        *count += 1;
    }
    for level in 1..=5 {
        assert_eq!(get_count_for_level(max, level) <= *count_per_level.get(&level).unwrap(), true)
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
