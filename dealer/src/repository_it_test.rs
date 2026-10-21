use std::path::{Path, PathBuf};

use tokio::sync::{Mutex, MutexGuard};
use tokio_postgres::error::SqlState;
use tokio_postgres::NoTls;
use tonic::{Code, Request, Status};
use uuid::{uuid, Uuid};

use super::pb::dealer_server::Dealer as _;
use super::pb::{DealRequest, ReviewAnswersRequest, StoreAnswersRequest};
use super::{CourseDecks, Dealer, Repository};

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

// Fake course service: returns the preset deck lists, or `error` when set.
#[derive(Default)]
struct FakeCourses {
    course_decks: Vec<Uuid>,
    lesson_decks: Vec<Uuid>,
    error: Option<Code>,
}

#[tonic::async_trait]
impl CourseDecks for FakeCourses {
    async fn course_deck_ids(&self, _course_id: &Uuid) -> Result<Vec<Uuid>, Status> {
        match self.error {
            Some(code) => Err(Status::new(code, "fake course service error")),
            None => Ok(self.course_decks.clone()),
        }
    }

    async fn lesson_deck_ids(&self, _lesson_id: &Uuid) -> Result<Vec<Uuid>, Status> {
        match self.error {
            Some(code) => Err(Status::new(code, "fake course service error")),
            None => Ok(self.lesson_decks.clone()),
        }
    }
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
        .get_max_cards_per_level(&USER, &[DECK], &10)
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
        .get_max_cards_per_level(&USER, &[DECK], &10)
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
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    assert!(actual.is_empty());
}

#[tokio::test]
async fn get_max_cards_per_level_uses_own_level() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
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
        .get_max_cards_per_level(&USER, &[DECK], &10)
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
        .get_max_cards_per_level(&USER, &[DECK], &10)
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
        .get_max_cards_per_level(&USER, &[DECK], &1)
        .await
        .unwrap();
    repo.client.execute("SELECT setseed(0.4)", &[]).await.unwrap();
    let second = repo
        .get_max_cards_per_level(&USER, &[DECK], &1)
        .await
        .unwrap();

    assert_eq!(1, first.len());
    assert_eq!(1, second.len());
    assert_ne!(first[0].id, second[0].id);
}

#[tokio::test]
async fn get_max_cards_per_level_limits_each_level_and_orders_by_level() {
    let (repo, _db_lock) = new_repository().await;
    let card_c = uuid!("cccccccc-0000-0000-0000-000000000000");
    let card_d = uuid!("dddddddd-0000-0000-0000-00000000000d");
    repo.client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $3, 'c'), ($2, $3, 'd')",
            &[&card_c, &card_d, &DECK],
        )
        .await
        .unwrap();
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 1),
        set_level(&repo, &USER, &CARD_B, 1),
        set_level(&repo, &USER, &card_c, 1),
        set_level(&repo, &USER, &card_d, 3),
    );

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &2)
        .await
        .unwrap();

    let actual_levels: Vec<usize> = actual.iter().map(|c| c.lvl).collect();
    assert_eq!(vec![1, 1, 3], actual_levels);
    assert_eq!(card_d.to_string(), actual[2].id);
}

#[tokio::test]
async fn get_max_cards_per_level_returns_cards_from_every_requested_deck() {
    let (repo, _db_lock) = new_repository().await;
    let other_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let card_e = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    repo.client
        .execute(
            "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'other', 'other')",
            &[&other_deck, &USER],
        )
        .await
        .unwrap();
    repo.client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'e')",
            &[&card_e, &other_deck],
        )
        .await
        .unwrap();
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &card_e, 2),
    );

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK, other_deck], &10)
        .await
        .unwrap();

    let mut actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    actual_ids.sort();
    assert_eq!(vec![CARD_A.to_string(), card_e.to_string()], actual_ids);
}

#[tokio::test]
async fn get_max_cards_per_level_excludes_decks_not_requested() {
    let (repo, _db_lock) = new_repository().await;
    let other_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let card_e = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    repo.client
        .execute(
            "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'other', 'other')",
            &[&other_deck, &USER],
        )
        .await
        .unwrap();
    repo.client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'e')",
            &[&card_e, &other_deck],
        )
        .await
        .unwrap();
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &card_e, 2),
    );

    let actual = repo
        .get_max_cards_per_level(&USER, &[other_deck], &10)
        .await
        .unwrap();

    let actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    assert_eq!(vec![card_e.to_string()], actual_ids);
}

#[tokio::test]
async fn get_max_cards_per_level_without_decks_returns_answered_cards_from_all_decks() {
    let (repo, _db_lock) = new_repository().await;
    let other_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let card_e = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    repo.client
        .execute(
            "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'other', 'other')",
            &[&other_deck, &USER],
        )
        .await
        .unwrap();
    repo.client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'e')",
            &[&card_e, &other_deck],
        )
        .await
        .unwrap();
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &card_e, 2),
    );

    let actual = repo
        .get_max_cards_per_level(&USER, &[], &10)
        .await
        .unwrap();

    let mut actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    actual_ids.sort();
    assert_eq!(vec![CARD_A.to_string(), card_e.to_string()], actual_ids);
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
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

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
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .store_answers(Request::new(StoreAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec!["99999999-0000-0000-0000-000000000000".to_owned()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::InvalidArgument, actual.code());
}

#[tokio::test]
async fn review_answers_correct_answer_raises_level_of_card_in_pool() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_CORRECT.to_string()],
        }))
        .await
        .unwrap();

    assert_eq!(3, level(&dealer.repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn review_answers_wrong_answer_lowers_level_of_card_in_pool() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 3).await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_WRONG.to_string()],
        }))
        .await
        .unwrap();

    assert_eq!(2, level(&dealer.repo, &USER, &CARD_A).await);
}

#[tokio::test]
async fn review_answers_rejects_card_outside_pool_and_writes_nothing() {
    let (repo, _db_lock) = new_repository().await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_CORRECT.to_string()],
        }))
        .await
        .unwrap_err();

    let practiced: i64 = dealer
        .repo
        .client
        .query_one("SELECT count(*) FROM card_practice", &[])
        .await
        .unwrap()
        .get(0);
    let leveled: i64 = dealer
        .repo
        .client
        .query_one("SELECT count(*) FROM user_card_level", &[])
        .await
        .unwrap()
        .get(0);
    assert_eq!(Code::InvalidArgument, actual.code());
    assert_eq!(0, practiced);
    assert_eq!(0, leveled);
}

#[tokio::test]
async fn review_answers_rejects_whole_batch_when_one_card_is_outside_pool() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_CORRECT.to_string(), ANSWER_B_CORRECT.to_string()],
        }))
        .await
        .unwrap_err();

    let practiced: i64 = dealer
        .repo
        .client
        .query_one("SELECT count(*) FROM card_practice", &[])
        .await
        .unwrap()
        .get(0);
    let leveled: i64 = dealer
        .repo
        .client
        .query_one("SELECT count(*) FROM user_card_level", &[])
        .await
        .unwrap()
        .get(0);
    assert_eq!(Code::InvalidArgument, actual.code());
    assert_eq!(2, level(&dealer.repo, &USER, &CARD_A).await);
    assert_eq!(0, practiced);
    assert_eq!(1, leveled);
}

#[tokio::test]
async fn review_answers_rejects_card_only_in_another_users_pool() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &OTHER_USER, &CARD_A, 2).await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_CORRECT.to_string()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::InvalidArgument, actual.code());
    assert_eq!(2, level(&dealer.repo, &OTHER_USER, &CARD_A).await);
}

#[tokio::test]
async fn review_answers_rejects_unknown_answer_id() {
    let (repo, _db_lock) = new_repository().await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec!["99999999-0000-0000-0000-000000000000".to_owned()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::InvalidArgument, actual.code());
}

#[tokio::test]
async fn deal_course_scope_deals_only_from_the_course_decks() {
    let (repo, _db_lock) = new_repository().await;
    let course_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let course_card = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    repo.client
        .execute(
            "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'course', 'course')",
            &[&course_deck, &USER],
        )
        .await
        .unwrap();
    repo.client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'course card')",
            &[&course_card, &course_deck],
        )
        .await
        .unwrap();
    // CARD_A is answered too but its deck isn't in the course, so it must not be dealt.
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &course_card, 2),
    );
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses {
            course_decks: vec![course_deck],
            ..Default::default()
        }),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            course_id: uuid!("c0c0c0c0-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap();

    assert_eq!(vec![course_card.to_string()], actual.get_ref().card_ids);
}

#[tokio::test]
async fn deal_lesson_scope_deals_only_from_the_lesson_decks() {
    let (repo, _db_lock) = new_repository().await;
    let outside_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let outside_card = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    repo.client
        .execute(
            "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'outside', 'outside')",
            &[&outside_deck, &USER],
        )
        .await
        .unwrap();
    repo.client
        .execute(
            "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'outside card')",
            &[&outside_card, &outside_deck],
        )
        .await
        .unwrap();
    // The lesson only has DECK; outside_card is answered too but must not be dealt.
    tokio::join!(
        set_level(&repo, &USER, &CARD_A, 2),
        set_level(&repo, &USER, &outside_card, 2),
    );
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses {
            lesson_decks: vec![DECK],
            ..Default::default()
        }),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            lesson_id: uuid!("1e1e1e1e-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap();

    assert_eq!(vec![CARD_A.to_string()], actual.get_ref().card_ids);
}

#[tokio::test]
async fn deal_course_without_decks_deals_nothing() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            course_id: uuid!("c0c0c0c0-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap();

    assert!(actual.get_ref().card_ids.is_empty());
}

#[tokio::test]
async fn deal_lesson_without_decks_deals_nothing() {
    let (repo, _db_lock) = new_repository().await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            lesson_id: uuid!("1e1e1e1e-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap();

    assert!(actual.get_ref().card_ids.is_empty());
}

#[tokio::test]
async fn deal_unknown_course_is_not_found() {
    let (repo, _db_lock) = new_repository().await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses {
            error: Some(Code::NotFound),
            ..Default::default()
        }),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            course_id: uuid!("c0c0c0c0-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::NotFound, actual.code());
}

#[tokio::test]
async fn deal_course_invalid_argument_is_returned_unchanged() {
    let (repo, _db_lock) = new_repository().await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses {
            error: Some(Code::InvalidArgument),
            ..Default::default()
        }),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            course_id: uuid!("c0c0c0c0-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::InvalidArgument, actual.code());
}

#[tokio::test]
async fn deal_course_service_failure_is_internal() {
    let (repo, _db_lock) = new_repository().await;
    let dealer = Dealer {
        repo,
        courses: Box::new(FakeCourses {
            error: Some(Code::Unavailable),
            ..Default::default()
        }),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            course_id: uuid!("c0c0c0c0-0000-0000-0000-000000000000").to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::Internal, actual.code());
}
