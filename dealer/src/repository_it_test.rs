use std::time::Duration;

use sqlx::postgres::{PgConnectOptions, PgPool, PgPoolOptions};
use tonic::{Code, Request, Status};
use uuid::{uuid, Uuid};

use super::pb::dealer_server::Dealer as _;
use super::pb::{DealRequest, ReviewAnswersRequest, StoreAnswersRequest};
use super::{new_pool, CourseDecks, Dealer, Repository, DB_TIMEOUT};

const USER: Uuid = uuid!("11111111-0000-0000-0000-000000000000");
const OTHER_USER: Uuid = uuid!("22222222-0000-0000-0000-000000000000");
const DECK: Uuid = uuid!("dddddddd-0000-0000-0000-000000000000");
const CARD_A: Uuid = uuid!("aaaaaaaa-0000-0000-0000-000000000000");
const CARD_B: Uuid = uuid!("bbbbbbbb-0000-0000-0000-000000000000");
const ANSWER_A_CORRECT: Uuid = uuid!("aaaaaaaa-0000-0000-0000-00000000000c");
const ANSWER_A_WRONG: Uuid = uuid!("aaaaaaaa-0000-0000-0000-00000000000e");
const ANSWER_B_CORRECT: Uuid = uuid!("bbbbbbbb-0000-0000-0000-00000000000c");

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

// Seeds the deck, cards and answers every test uses. #[sqlx::test] gives each test its own
// database with the deck migrations applied.
async fn new_repository(pool: PgPool) -> Repository {
    sqlx::query!(
        "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'deck', 'deck')",
        DECK,
        USER,
    )
    .execute(&pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $3, 'a'), ($2, $3, 'b')",
        CARD_A,
        CARD_B,
        DECK,
    )
    .execute(&pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO answers (id, card_id, text, is_correct) VALUES
            ($1, $4, 'right', true),
            ($2, $4, 'wrong', false),
            ($3, $5, 'right', true)",
        ANSWER_A_CORRECT,
        ANSWER_A_WRONG,
        ANSWER_B_CORRECT,
        CARD_A,
        CARD_B,
    )
    .execute(&pool)
    .await
    .unwrap();

    Repository { pool }
}

async fn set_level(repo: &Repository, user_id: &Uuid, card_id: &Uuid, lvl: i32) {
    sqlx::query!(
        "INSERT INTO user_card_level (user_id, card_id, lvl, updated_at) VALUES ($1, $2, $3, now())",
        user_id,
        card_id,
        lvl
    )
    .execute(&repo.pool)
    .await
    .unwrap();
}

async fn level(repo: &Repository, user_id: &Uuid, card_id: &Uuid) -> i32 {
    sqlx::query_scalar!(
        "SELECT lvl FROM user_card_level WHERE user_id = $1 AND card_id = $2",
        user_id,
        card_id
    )
    .fetch_one(&repo.pool)
    .await
    .unwrap()
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_first_correct_answer_raises_level_to_2(pool: PgPool) {
    let repo = new_repository(pool).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT])
        .await
        .unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_correct_answer_raises_level(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT])
        .await
        .unwrap();

    assert_eq!(4, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_correct_answer_caps_level_at_5(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 5).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT])
        .await
        .unwrap();

    assert_eq!(5, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_wrong_answer_lowers_level(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_first_wrong_answer_keeps_level_at_1(pool: PgPool) {
    let repo = new_repository(pool).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    assert_eq!(1, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_wrong_answer_keeps_existing_level_1(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 1).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    assert_eq!(1, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_right_and_wrong_for_same_card_counts_as_wrong(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, ANSWER_A_WRONG])
        .await
        .unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_ignores_other_users_level(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &OTHER_USER, &CARD_A, 4).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT])
        .await
        .unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
    assert_eq!(4, level(&repo, &OTHER_USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_updates_each_card_independently(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG, ANSWER_B_CORRECT])
        .await
        .unwrap();

    assert_eq!(2, level(&repo, &USER, &CARD_A).await);
    assert_eq!(2, level(&repo, &USER, &CARD_B).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_records_every_answer_in_card_practice(pool: PgPool) {
    let repo = new_repository(pool).await;

    repo.store_answers(&USER, &[ANSWER_A_WRONG, ANSWER_B_CORRECT])
        .await
        .unwrap();

    let actual: i64 = sqlx::query_scalar!(
        "SELECT count(*) AS \"count!\" FROM card_practice WHERE user_id = $1",
        USER
    )
    .fetch_one(&repo.pool)
    .await
    .unwrap();
    assert_eq!(2, actual);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_unknown_answer_is_foreign_key_violation(pool: PgPool) {
    let repo = new_repository(pool).await;

    let actual = repo
        .store_answers(&USER, &[uuid!("99999999-0000-0000-0000-000000000000")])
        .await
        .unwrap_err();

    assert!(super::is_unknown_answer(&actual));
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_unknown_answer_rolls_back_known_answers(pool: PgPool) {
    let repo = new_repository(pool).await;

    repo.store_answers(
        &USER,
        &[
            ANSWER_A_CORRECT,
            uuid!("99999999-0000-0000-0000-000000000000"),
        ],
    )
    .await
    .unwrap_err();

    let actual_practiced: i64 =
        sqlx::query_scalar!("SELECT count(*) AS \"count!\" FROM card_practice")
            .fetch_one(&repo.pool)
            .await
            .unwrap();
    let actual_leveled: i64 =
        sqlx::query_scalar!("SELECT count(*) AS \"count!\" FROM user_card_level")
            .fetch_one(&repo.pool)
            .await
            .unwrap();
    assert_eq!(0, actual_practiced);
    assert_eq!(0, actual_leveled);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_ignores_other_users_levels(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &OTHER_USER, &CARD_A, 4).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    assert_eq!(1, actual.len());
    assert_eq!(CARD_A.to_string(), actual[0].id);
    assert_eq!(2, actual[0].lvl);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_excludes_never_answered_cards(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 2).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    let actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    assert_eq!(vec![CARD_A.to_string()], actual_ids);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_excludes_cards_only_other_users_answered(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &OTHER_USER, &CARD_A, 4).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    assert!(actual.is_empty());
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_uses_own_level(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    let card_a = actual.iter().find(|c| c.id == CARD_A.to_string()).unwrap();
    assert_eq!(3, card_a.lvl);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_excludes_deleted_cards(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &CARD_B, 2).await;
    sqlx::query!("UPDATE cards SET deleted_at = now() WHERE id = $1", CARD_B)
        .execute(&repo.pool)
        .await
        .unwrap();

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    let actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    assert_eq!(vec![CARD_A.to_string()], actual_ids);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_excludes_cards_of_deleted_deck(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &CARD_B, 2).await;
    sqlx::query!("UPDATE decks SET deleted_at = now() WHERE id = $1", DECK)
        .execute(&repo.pool)
        .await
        .unwrap();

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &10)
        .await
        .unwrap();

    assert!(actual.is_empty());
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_picks_randomly_within_a_level(
    _: PgPoolOptions,
    options: PgConnectOptions,
) {
    // One connection, so setseed applies to the query that follows it.
    let repo = new_repository(new_pool(options, 1, DB_TIMEOUT)).await;
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &CARD_B, 2).await;

    sqlx::query("SELECT setseed(0.1)")
        .execute(&repo.pool)
        .await
        .unwrap();
    let actual_first = repo
        .get_max_cards_per_level(&USER, &[DECK], &1)
        .await
        .unwrap();
    sqlx::query("SELECT setseed(0.4)")
        .execute(&repo.pool)
        .await
        .unwrap();
    let actual_second = repo
        .get_max_cards_per_level(&USER, &[DECK], &1)
        .await
        .unwrap();

    assert_eq!(1, actual_first.len());
    assert_eq!(1, actual_second.len());
    assert_ne!(actual_first[0].id, actual_second[0].id);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_limits_each_level_and_orders_by_level(pool: PgPool) {
    let repo = new_repository(pool).await;
    let card_c = uuid!("cccccccc-0000-0000-0000-000000000000");
    let card_d = uuid!("dddddddd-0000-0000-0000-00000000000d");
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $3, 'c'), ($2, $3, 'd')",
        card_c,
        card_d,
        DECK
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    set_level(&repo, &USER, &CARD_A, 1).await;
    set_level(&repo, &USER, &CARD_B, 1).await;
    set_level(&repo, &USER, &card_c, 1).await;
    set_level(&repo, &USER, &card_d, 3).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK], &2)
        .await
        .unwrap();

    let actual_levels: Vec<usize> = actual.iter().map(|c| c.lvl).collect();
    assert_eq!(vec![1, 1, 3], actual_levels);
    assert_eq!(card_d.to_string(), actual[2].id);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_returns_cards_from_every_requested_deck(pool: PgPool) {
    let repo = new_repository(pool).await;
    let other_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let card_e = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    sqlx::query!(
        "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'other', 'other')",
        other_deck,
        USER
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'e')",
        card_e,
        other_deck
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &card_e, 2).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[DECK, other_deck], &10)
        .await
        .unwrap();

    let mut actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    actual_ids.sort();
    assert_eq!(vec![CARD_A.to_string(), card_e.to_string()], actual_ids);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_excludes_decks_not_requested(pool: PgPool) {
    let repo = new_repository(pool).await;
    let other_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let card_e = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    sqlx::query!(
        "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'other', 'other')",
        other_deck,
        USER
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'e')",
        card_e,
        other_deck
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &card_e, 2).await;

    let actual = repo
        .get_max_cards_per_level(&USER, &[other_deck], &10)
        .await
        .unwrap();

    let actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    assert_eq!(vec![card_e.to_string()], actual_ids);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn get_max_cards_per_level_without_decks_returns_answered_cards_from_all_decks(pool: PgPool) {
    let repo = new_repository(pool).await;
    let other_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let card_e = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    sqlx::query!(
        "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'other', 'other')",
        other_deck,
        USER
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'e')",
        card_e,
        other_deck
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &card_e, 2).await;

    let actual = repo.get_max_cards_per_level(&USER, &[], &10).await.unwrap();

    let mut actual_ids: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    actual_ids.sort();
    assert_eq!(vec![CARD_A.to_string(), card_e.to_string()], actual_ids);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_correct_answer_refreshes_updated_at(pool: PgPool) {
    let repo = new_repository(pool).await;
    sqlx::query!(
        "INSERT INTO user_card_level (user_id, card_id, lvl, updated_at)
            VALUES ($1, $2, 3, now() - interval '30 days')",
        USER,
        CARD_A
    )
    .execute(&repo.pool)
    .await
    .unwrap();

    repo.store_answers(&USER, &[ANSWER_A_CORRECT])
        .await
        .unwrap();

    let actual: bool = sqlx::query_scalar!(
        "SELECT updated_at > now() - interval '1 minute' AS \"recent!\"
            FROM user_card_level WHERE user_id = $1 AND card_id = $2",
        USER,
        CARD_A
    )
    .fetch_one(&repo.pool)
    .await
    .unwrap();
    assert!(actual);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_wrong_answer_refreshes_updated_at(pool: PgPool) {
    let repo = new_repository(pool).await;
    sqlx::query!(
        "INSERT INTO user_card_level (user_id, card_id, lvl, updated_at)
            VALUES ($1, $2, 3, now() - interval '30 days')",
        USER,
        CARD_A
    )
    .execute(&repo.pool)
    .await
    .unwrap();

    repo.store_answers(&USER, &[ANSWER_A_WRONG]).await.unwrap();

    let actual: bool = sqlx::query_scalar!(
        "SELECT updated_at > now() - interval '1 minute' AS \"recent!\"
            FROM user_card_level WHERE user_id = $1 AND card_id = $2",
        USER,
        CARD_A
    )
    .fetch_one(&repo.pool)
    .await
    .unwrap();
    assert!(actual);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_duplicate_answer_raises_level_once(pool: PgPool) {
    let repo = new_repository(pool).await;
    set_level(&repo, &USER, &CARD_A, 3).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, ANSWER_A_CORRECT])
        .await
        .unwrap();

    assert_eq!(4, level(&repo, &USER, &CARD_A).await);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_duplicate_answer_is_recorded_twice(pool: PgPool) {
    let repo = new_repository(pool).await;

    repo.store_answers(&USER, &[ANSWER_A_CORRECT, ANSWER_A_CORRECT])
        .await
        .unwrap();

    let actual: i64 = sqlx::query_scalar!(
        "SELECT count(*) AS \"count!\" FROM card_practice WHERE user_id = $1",
        USER
    )
    .fetch_one(&repo.pool)
    .await
    .unwrap();
    assert_eq!(2, actual);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_rejects_answer_without_card(pool: PgPool) {
    let repo = new_repository(pool).await;
    let orphan_answer = uuid!("cccccccc-0000-0000-0000-00000000000c");
    sqlx::query!(
        "INSERT INTO answers (id, card_id, text, is_correct) VALUES ($1, NULL, 'orphan', true)",
        orphan_answer
    )
    .execute(&repo.pool)
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn store_answers_rejects_unknown_answer_id(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn review_answers_correct_answer_raises_level_of_card_in_pool(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn review_answers_wrong_answer_lowers_level_of_card_in_pool(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn review_answers_rejects_card_outside_pool_and_writes_nothing(pool: PgPool) {
    let repo = new_repository(pool).await;
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

    let actual_practiced: i64 =
        sqlx::query_scalar!("SELECT count(*) AS \"count!\" FROM card_practice")
            .fetch_one(&dealer.repo.pool)
            .await
            .unwrap();
    let actual_leveled: i64 =
        sqlx::query_scalar!("SELECT count(*) AS \"count!\" FROM user_card_level")
            .fetch_one(&dealer.repo.pool)
            .await
            .unwrap();
    assert_eq!(Code::InvalidArgument, actual.code());
    assert_eq!(0, actual_practiced);
    assert_eq!(0, actual_leveled);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn review_answers_rejects_whole_batch_when_one_card_is_outside_pool(pool: PgPool) {
    let repo = new_repository(pool).await;
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

    let actual_practiced: i64 =
        sqlx::query_scalar!("SELECT count(*) AS \"count!\" FROM card_practice")
            .fetch_one(&dealer.repo.pool)
            .await
            .unwrap();
    let actual_leveled: i64 =
        sqlx::query_scalar!("SELECT count(*) AS \"count!\" FROM user_card_level")
            .fetch_one(&dealer.repo.pool)
            .await
            .unwrap();
    assert_eq!(Code::InvalidArgument, actual.code());
    assert_eq!(2, level(&dealer.repo, &USER, &CARD_A).await);
    assert_eq!(0, actual_practiced);
    assert_eq!(1, actual_leveled);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn review_answers_rejects_card_only_in_another_users_pool(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn review_answers_rejects_unknown_answer_id(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_course_scope_deals_only_from_the_course_decks(pool: PgPool) {
    let repo = new_repository(pool).await;
    let course_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let course_card = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    sqlx::query!(
        "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'course', 'course')",
        course_deck,
        USER
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'course card')",
        course_card,
        course_deck
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    // CARD_A is answered too but its deck isn't in the course, so it must not be dealt.
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &course_card, 2).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_lesson_scope_deals_only_from_the_lesson_decks(pool: PgPool) {
    let repo = new_repository(pool).await;
    let outside_deck = uuid!("eeeeeeee-0000-0000-0000-000000000000");
    let outside_card = uuid!("eeeeeeee-0000-0000-0000-00000000000e");
    sqlx::query!(
        "INSERT INTO decks (id, author_id, title, description) VALUES ($1, $2, 'outside', 'outside')",
        outside_deck,
        USER
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    sqlx::query!(
        "INSERT INTO cards (id, deck_id, title) VALUES ($1, $2, 'outside card')",
        outside_card,
        outside_deck
    )
    .execute(&repo.pool)
    .await
    .unwrap();
    // The lesson only has DECK; outside_card is answered too but must not be dealt.
    set_level(&repo, &USER, &CARD_A, 2).await;
    set_level(&repo, &USER, &outside_card, 2).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_course_without_decks_deals_nothing(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_lesson_without_decks_deals_nothing(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_unknown_course_is_not_found(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_course_invalid_argument_is_returned_unchanged(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn deal_course_service_failure_is_internal(pool: PgPool) {
    let repo = new_repository(pool).await;
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

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn repository_recovers_after_its_database_connection_is_killed(
    _: PgPoolOptions,
    options: PgConnectOptions,
) {
    let repo = new_repository(new_pool(options.clone(), 1, DB_TIMEOUT)).await;
    set_level(&repo, &USER, &CARD_A, 2).await;

    // The repository's pool has one connection, so this is the backend to end. A separate pool
    // ends it, waiting up to 5 s for it to close.
    let pid: i32 = sqlx::query_scalar("SELECT pg_backend_pid()")
        .fetch_one(&repo.pool)
        .await
        .unwrap();
    let killer = new_pool(options, 1, DB_TIMEOUT);
    sqlx::query("SELECT pg_terminate_backend($1, 5000)")
        .bind(pid)
        .execute(&killer)
        .await
        .unwrap();

    let actual = repo.get_max_cards_per_level(&USER, &[], &10).await.unwrap();

    let actual: Vec<String> = actual.iter().map(|c| c.id.clone()).collect();
    let expected = vec![CARD_A.to_string()];
    assert_eq!(expected, actual);
}

#[sqlx::test(migrations = "../deck/cmd/migrate/migrations")]
async fn query_running_longer_than_the_timeout_is_cancelled(
    _: PgPoolOptions,
    options: PgConnectOptions,
) {
    let pool = new_pool(options, 1, Duration::from_secs(1));

    let actual = sqlx::query("SELECT pg_sleep(5)")
        .execute(&pool)
        .await
        .unwrap_err();

    let actual = actual
        .as_database_error()
        .and_then(|err| err.code())
        .map(|code| code.into_owned());
    // 57014 is Postgres' query_canceled.
    assert_eq!(Some("57014".to_owned()), actual);
}

#[tokio::test]
async fn store_answers_without_a_database_is_internal() {
    let nothing_listening = PgConnectOptions::new().host("localhost").port(1);
    let dealer = Dealer {
        repo: Repository {
            pool: new_pool(nothing_listening, 1, Duration::from_millis(200)),
        },
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .store_answers(Request::new(StoreAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_CORRECT.to_string()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::Internal, actual.code());
}

#[tokio::test]
async fn review_answers_without_a_database_is_internal() {
    let nothing_listening = PgConnectOptions::new().host("localhost").port(1);
    let dealer = Dealer {
        repo: Repository {
            pool: new_pool(nothing_listening, 1, Duration::from_millis(200)),
        },
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .review_answers(Request::new(ReviewAnswersRequest {
            user_id: USER.to_string(),
            answer_ids: vec![ANSWER_A_CORRECT.to_string()],
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::Internal, actual.code());
}

#[tokio::test]
async fn deal_without_a_database_is_internal() {
    let nothing_listening = PgConnectOptions::new().host("localhost").port(1);
    let dealer = Dealer {
        repo: Repository {
            pool: new_pool(nothing_listening, 1, Duration::from_millis(200)),
        },
        courses: Box::new(FakeCourses::default()),
    };

    let actual = dealer
        .deal(Request::new(DealRequest {
            user_id: USER.to_string(),
            number_of_cards: 10,
            ..Default::default()
        }))
        .await
        .unwrap_err();

    assert_eq!(Code::Internal, actual.code());
}
