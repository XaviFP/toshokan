use std::path::{Path, PathBuf};

use tokio::sync::{Mutex, MutexGuard};
use tokio_postgres::error::SqlState;
use tokio_postgres::NoTls;
use tonic::{Code, Request};
use uuid::{uuid, Uuid};

use super::pb::dealer_server::Dealer as _;
use super::pb::StoreAnswersRequest;
use super::{Dealer, Repository};

const USER: Uuid = uuid!("11111111-0000-0000-0000-000000000000");
const OTHER_USER: Uuid = uuid!("22222222-0000-0000-0000-000000000000");
const DECK: Uuid = uuid!("dddddddd-0000-0000-0000-000000000000");
const CARD_A: Uuid = uuid!("aaaaaaaa-0000-0000-0000-000000000000");
const CARD_B: Uuid = uuid!("bbbbbbbb-0000-0000-0000-000000000000");
const ANSWER_A_CORRECT: Uuid = uuid!("aaaaaaaa-0000-0000-0000-00000000000c");
const ANSWER_A_WRONG: Uuid = uuid!("aaaaaaaa-0000-0000-0000-00000000000e");
const ANSWER_B_CORRECT: Uuid = uuid!("bbbbbbbb-0000-0000-0000-00000000000c");

fn deck_migrations() -> Vec<String> {
    let dir = Path::new(env!("CARGO_MANIFEST_DIR")).join("../deck/cmd/migrate/migrations");
    let mut files: Vec<PathBuf> = std::fs::read_dir(dir)
        .unwrap()
        .map(|entry| entry.unwrap().path())
        .filter(|path| path.to_string_lossy().ends_with(".up.sql"))
        .collect();
    files.sort_by_key(|path| {
        let name = path.file_name().unwrap().to_string_lossy().into_owned();
        name.split('_').next().unwrap().parse::<u32>().unwrap()
    });
    files.iter().map(|path| std::fs::read_to_string(path).unwrap()).collect()
}

static DB_LOCK: Mutex<()> = Mutex::const_new(());

// Resets the test_dealer database; the returned guard keeps other tests out until this one ends.
async fn new_repository() -> (Repository, MutexGuard<'static, ()>) {
    let db_lock = DB_LOCK.lock().await;
    let (client, connection) = tokio_postgres::connect(
        "host=localhost port=5432 user=toshokan password=t.o.s.h.o.k.a.n. dbname=test_dealer",
        NoTls,
    )
    .await
    .unwrap();
    tokio::spawn(async move {
        if let Err(err) = connection.await {
            eprintln!("test database connection error: {}", err);
        }
    });

    client
        .batch_execute("DROP SCHEMA IF EXISTS public CASCADE; CREATE SCHEMA public;")
        .await
        .unwrap();
    for migration in deck_migrations() {
        client.batch_execute(&migration).await.unwrap();
    }

    client
        .execute(
            "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'deck', 'deck')",
            &[&DECK, &USER],
        )
        .await
        .unwrap();
    client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $3, 'a'), ($2, $3, 'b')",
            &[&CARD_A, &CARD_B, &DECK],
        )
        .await
        .unwrap();
    client
        .execute(
            "INSERT INTO answers (id, card_id, text, is_correct) VALUES
                ($1, $4, 'right', true),
                ($2, $4, 'wrong', false),
                ($3, $5, 'right', true)",
            &[&ANSWER_A_CORRECT, &ANSWER_A_WRONG, &ANSWER_B_CORRECT, &CARD_A, &CARD_B],
        )
        .await
        .unwrap();

    (Repository { client }, db_lock)
}

async fn set_level(repo: &Repository, user_id: &Uuid, card_id: &Uuid, lvl: i32) {
    repo.client
        .execute(
            "INSERT INTO user_card_level (user_id, card_id, lvl, updated_at) VALUES ($1, $2, $3, now())",
            &[user_id, card_id, &lvl],
        )
        .await
        .unwrap();
}

async fn level(repo: &Repository, user_id: &Uuid, card_id: &Uuid) -> i32 {
    repo.client
        .query_one(
            "SELECT lvl FROM user_card_level WHERE user_id = $1 AND card_id = $2",
            &[user_id, card_id],
        )
        .await
        .unwrap()
        .get(0)
}

#[tokio::test]
async fn store_answers_first_correct_answer_raises_level_to_2() {
    let (repo, _db_lock) = new_repository().await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT]).await.unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_correct_answer_raises_level() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT]).await.unwrap();

    assert_eq!(4, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_correct_answer_caps_level_at_5() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 5).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT]).await.unwrap();

    assert_eq!(5, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_wrong_answer_lowers_level() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_first_wrong_answer_keeps_level_at_1() {
    let (repo, _db_lock) = new_repository().await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    assert_eq!(1, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_wrong_answer_keeps_existing_level_1() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 1).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    assert_eq!(1, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_right_and_wrong_for_same_card_counts_as_wrong() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, ANSWER_A_WRONG])
        .await
        .unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_ignores_other_users_level() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &OTHER_USER, &CARD_A, 4).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT]).await.unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
    assert_eq!(4, level(&repo, &OTHER_USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_updates_each_card_independently() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG, ANSWER_B_CORRECT])
        .await
        .unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
    assert_eq!(2, level(&repo, &USER, &CARD_B).await);
}

#[tokio::test]
async fn store_answers_records_every_answer_in_card_practice() {
    let (repo, _db_lock) = new_repository().await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG, ANSWER_B_CORRECT])
        .await
        .unwrap();

    let actual: i64 = repo
        .client
        .query_one("SELECT count(*) FROM card_practice WHERE user_id = $1", &[&USER])
        .await
        .unwrap()
        .get(0);
    assert_eq!(2, actual);
}

#[tokio::test]
async fn store_answers_unknown_answer_is_foreign_key_violation() {
    let (repo, _db_lock) = new_repository().await;

    let actual = repo
        .store_answers(&USER, &[uuid!("99999999-0000-0000-0000-000000000000")])
        .await
        .unwrap_err();

    assert_eq!(Some(&SqlState::FOREIGN_KEY_VIOLATION), actual.code());
}

#[tokio::test]
async fn store_answers_unknown_answer_rolls_back_known_answers() {
    let (repo, _db_lock) = new_repository().await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, uuid!("99999999-0000-0000-0000-000000000000")])
        .await
        .unwrap_err();

    let practiced: i64 = repo
        .client
        .query_one("SELECT count(*) FROM card_practice", &[])
        .await
        .unwrap()
        .get(0);
    let leveled: i64 = repo
        .client
        .query_one("SELECT count(*) FROM user_card_level", &[])
        .await
        .unwrap()
        .get(0);
    assert_eq!(0, practiced);
    assert_eq!(0, leveled);
}

#[tokio::test]
async fn get_max_cards_per_level_ignores_other_users_levels() {
    let (repo, _db_lock) = new_repository().await;
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &OTHER_USER, &CARD_A, 4),
    );

    let actual = repo
        .get_max_cards_per_level(&USER, &DECK, &10)
        .await
        .unwrap();

    assert_eq!(1, actual.len());
    assert_eq!(CARD_A.to_string(), actual[0].id);
    assert_eq!(2, actual[0].lvl);
}

#[tokio::test]
async fn get_max_cards_per_level_excludes_never_answered_cards() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 2).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &DECK, &10)
        .await
        .unwrap();

    let ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    assert_eq!(vec![CARD_A.to_string()], ids);
}

#[tokio::test]
async fn get_max_cards_per_level_excludes_cards_only_other_users_answered() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &OTHER_USER, &CARD_A, 4).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &DECK, &10)
        .await
        .unwrap();

    assert!(actual.is_empty());
}

#[tokio::test]
async fn get_max_cards_per_level_uses_own_level() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &DECK, &10)
        .await
        .unwrap();

    let card_a = actual.iter().find(|c| c.id == CARD_A.to_string()).unwrap();
    assert_eq!(3, card_a.lvl);
}

#[tokio::test]
async fn get_max_cards_per_level_excludes_deleted_cards() {
    let (repo, _db_lock) = new_repository().await;
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &CARD_B, 2),
    );
    repo.client
        .execute("UPDATE cards SET deleted_at = now() WHERE id = $1", &[&CARD_B])
        .await
        .unwrap();

    let actual = repo
        .get_max_cards_per_level(&USER, &DECK, &10)
        .await
        .unwrap();

    let ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    assert_eq!(vec![CARD_A.to_string()], ids);
}

#[tokio::test]
async fn get_max_cards_per_level_excludes_cards_of_deleted_deck() {
    let (repo, _db_lock) = new_repository().await;
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &CARD_B, 2),
    );
    repo.client
        .execute("UPDATE decks SET deleted_at = now() WHERE id = $1", &[&DECK])
        .await
        .unwrap();

    let actual = repo
        .get_max_cards_per_level(&USER, &DECK, &10)
        .await
        .unwrap();

    assert!(actual.is_empty());
}

#[tokio::test]
async fn get_max_cards_per_level_picks_randomly_within_a_level() {
    let (repo, _db_lock) = new_repository().await;
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &CARD_B, 2),
    );

    repo.client.execute("SELECT setseed(0.1)", &[]).await.unwrap();
    let first = repo
        .get_max_cards_per_level(&USER, &DECK, &1)
        .await
        .unwrap();
    repo.client.execute("SELECT setseed(0.4)", &[]).await.unwrap();
    let second = repo
        .get_max_cards_per_level(&USER, &DECK, &1)
        .await
        .unwrap();

    assert_eq!(1, first.len());
    assert_eq!(1, second.len());
    assert_ne!(first[0].id, second[0].id);
}

#[tokio::test]
async fn store_answers_correct_answer_refreshes_updated_at() {
    let (repo, _db_lock) = new_repository().await;
    repo.client
        .execute(
            "INSERT INTO user_card_level (user_id, card_id, lvl, updated_at) VALUES ($1, $2, 3, now() - interval '30 days')",
            &[&USER, &CARD_A],
        )
        .await
        .unwrap();

    repo.store_answers(&USER, &[ANSWER_A_CORRECT]).await.unwrap();

    let actual: bool = repo
        .client
        .query_one(
            "SELECT updated_at > now() - interval '1 minute' FROM user_card_level WHERE user_id = $1 AND card_id = $2",
            &[&USER, &CARD_A],
        )
        .await
        .unwrap()
        .get(0);
    assert!(actual);
}

#[tokio::test]
async fn store_answers_wrong_answer_refreshes_updated_at() {
    let (repo, _db_lock) = new_repository().await;
    repo.client
        .execute(
            "INSERT INTO user_card_level (user_id, card_id, lvl, updated_at) VALUES ($1, $2, 3, now() - interval '30 days')",
            &[&USER, &CARD_A],
        )
        .await
        .unwrap();

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    let actual: bool = repo
        .client
        .query_one(
            "SELECT updated_at > now() - interval '1 minute' FROM user_card_level WHERE user_id = $1 AND card_id = $2",
            &[&USER, &CARD_A],
        )
        .await
        .unwrap()
        .get(0);
    assert!(actual);
}

#[tokio::test]
async fn store_answers_duplicate_answer_raises_level_once() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, ANSWER_A_CORRECT])
        .await
        .unwrap();

    assert_eq!(4, level(&repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn store_answers_duplicate_answer_is_recorded_twice() {
    let (repo, _db_lock) = new_repository().await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, ANSWER_A_CORRECT])
        .await
        .unwrap();

    let actual: i64 = repo
        .client
        .query_one("SELECT count(*) FROM card_practice WHERE user_id = $1", &[&USER])
        .await
        .unwrap()
        .get(0);
    assert_eq!(2, actual);
}

#[tokio::test]
async fn store_answers_rejects_answer_without_card() {
    let (repo, _db_lock) = new_repository().await;
    let orphan_answer = uuid!("cccccccc-0000-0000-0000-00000000000c");
    repo.client
        .execute(
            "INSERT INTO answers (id, card_id, text, is_correct) VALUES ($1, NULL, 'orphan', true)",
            &[&orphan_answer],
        )
        .await
        .unwrap();
    let dealer = Dealer { repo };

    let actual = dealer
        .store_answers(Request::new(StoreAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![orphan_answer.to_string()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::InvalidArgument, actual.code());
}

#[tokio::test]
async fn store_answers_rejects_unknown_answer_id() {
    let (repo, _db_lock) = new_repository().await;
    let dealer = Dealer { repo };

    let actual = dealer
        .store_answers(Request::new(StoreAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec!["99999999-0000-0000-0000-000000000000".to_owned()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::InvalidArgument, actual.code());
}
