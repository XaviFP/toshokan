use std::collections::HashMap;
use std::fmt::Write;

use postgres::types::ToSql;
use postgres::NoTls;
use tokio::signal::unix::{signal, SignalKind};
use tokio_util::sync::CancellationToken;
use tonic::Code;
use tonic::{transport::Server, Request, Response, Status};
use uuid::Uuid;

mod pb {
    include!("../api/proto/v1/v1.rs");
}

use pb::dealer_server::{Dealer as pbDealer, DealerServer};
use pb::{DealRequest, DealResponse, StoreAnswersRequest, StoreAnswersResponse};

mod config;
use config::{load_db_config, load_grpc_server_config};

mod argumenter;
use argumenter::Argumenter;

mod ranker;
use ranker::Ranker;

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
                    SELECT cards.id as card_id, COALESCE(user_card_level.lvl, 1) as lvl
                    FROM cards
                    LEFT JOIN user_card_level
                    ON user_card_level.card_id = cards.id
                    WHERE (user_card_level.user_id = $1
                    OR user_card_level.card_id IS NULL)
                    AND cards.deck_id = $2
                )
                (SELECT card_id, lvl FROM user_cards WHERE lvl = 1 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 2 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 3 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 4 LIMIT $3)
                UNION (SELECT card_id, lvl FROM user_cards WHERE lvl = 5 LIMIT $3)
                ORDER BY lvl DESC",
                &[
                    &Uuid::parse_str(user_id).unwrap(),
                    &Uuid::parse_str(deck_id).unwrap(),
                    max,
                ],
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

    async fn store_answers(
        &self,
        user_id: &String,
        answer_ids: &Vec<String>,
    ) -> Result<(), String> {
        if answer_ids.len() == 0 {
            return Err("No answers provided".to_owned());
        }
        let mut buf = String::from("INSERT INTO card_practice (user_id, answer_id) VALUES ");
        let mut arger = Argumenter {
            values: Vec::<&(dyn ToSql + Sync)>::new(),
        };

        let uid_res = Uuid::try_parse(user_id);
        if uid_res.is_err() {
            return Err(uid_res.err().unwrap().to_string());
        }

        let uid = uid_res.unwrap();

        // Transform srings to valid UUIDs
        // Should fail or at least inform of invalid UUIDs received
        let ans_ids: Vec<Uuid> = answer_ids
            .iter()
            .filter_map(|answer| {
                let ans_res = Uuid::try_parse(answer);
                if ans_res.is_err() {
                    return None;
                }
                Some(ans_res.unwrap())
            })
            .collect();

        for answer in ans_ids.iter() {
            let res = buf.write_fmt(format_args!("({},{}),", arger.add(&uid), arger.add(answer)));
            if res.is_err() {
                return Err(res.err().unwrap().to_string());
            }
        }
        // Remove trailing comma
        buf.pop();
        let result = self.client.execute(&buf, arger.values()).await;
        if result.is_err() {
            return Err(result.err().unwrap().to_string());
        }

        // Update actual user_card_levels
        let mut arger = Argumenter {
            values: Vec::<&(dyn ToSql + Sync)>::new(),
        };
        let mut answers_string_arg = String::from("");

        for answer in ans_ids.iter() {
            let res = answers_string_arg.write_fmt(format_args!("{},", arger.add(answer)));
            if res.is_err() {
                return Err(res.err().unwrap().to_string());
            }
        }

        // Remove trailing comma
        answers_string_arg.pop();

        let result = self
            .client
            .execute(
                &format!(
                    "WITH correct_answered_cards as (
                    SELECT card_id FROM answers
                    WHERE is_correct = true
                    AND id in ({})
                )
                INSERT INTO user_card_level (user_id, card_id, updated_at)
                (
                    SELECT {} as user_id, c.card_id as card_id, now() as updated_at
                    FROM correct_answered_cards as c
                )
                ON CONFLICT ON CONSTRAINT user_card_level_card_id_user_id_key DO UPDATE
                SET
                lvl = LEAST((SELECT lvl FROM user_card_level WHERE user_id = EXCLUDED.user_id AND card_id = EXCLUDED.card_id) + 1, 5),
                updated_at = now()",
                    answers_string_arg,
                    arger.add(&uid),
                ),
                arger.values(),
            )
            .await;
        if result.is_err() {
            return Err(result.err().unwrap().to_string());
        }

        Ok(())
    }
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
        let result = self
            .repo
            .store_answers(&request.get_ref().user_id, &request.get_ref().answer_ids)
            .await;
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
        let max = i64::from(request.get_ref().number_of_cards);
        let result = self
            .repo
            .get_max_cards_per_level(&request.get_ref().user_id, &request.get_ref().deck_id, &max)
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

    let dealer = Dealer {
        repo: Repository { client },
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
