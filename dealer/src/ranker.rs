use chrono::Local;
use std::{env, time};
use tokio::time::{sleep, Duration};
use tokio_util::sync::CancellationToken;

pub struct Ranker {
    client: tokio_postgres::Client,
}

impl Ranker {
    pub fn new(client: tokio_postgres::Client) -> Self {
        Self { client }
    }

    pub async fn start(&self, cancel_token: CancellationToken) -> Result<(), String> {
        let r_config = load_ranker_config();
        if r_config.is_err() {
            eprintln!(
                "Ranker configuration error: {}",
                r_config.as_ref().err().unwrap()
            );
            return Err(r_config.err().unwrap());
        }
        let config = r_config.unwrap();
        let frequency: u64 = config.run_frequency.parse().unwrap();
        let period_num: u64 = config.period_elapsed.parse().unwrap();
        let period_res = chrono::Duration::from_std(time::Duration::from_secs(period_num));
        if period_res.is_err() {
            return Err(period_res.err().unwrap().to_string());
        }

        loop {
            let timestamp_to_compare = Local::now().checked_sub_signed(period_res.unwrap());
            if timestamp_to_compare.is_none() {
                return Err("Datetime formation is wrong".to_owned());
            }

            let result = self
                .client
                .execute(
                    "UPDATE user_card_level
                SET lvl = lvl - 1, updated_at = now()
                WHERE updated_at < $1
                AND lvl > 1",
                    &[&timestamp_to_compare],
                )
                .await;

            if result.is_err() {
                println!("{}", result.err().unwrap().to_string());
            }

            tokio::select! {
                _ = sleep(Duration::from_secs(frequency)) => {},
                _ = cancel_token.cancelled() => {
                    println!("Ranker received shutdown signal, stopping...");
                    return Ok(());
                }
            }
        }
    }
}

pub struct RankerConfig {
    pub run_frequency: String,
    pub period_elapsed: String,
}

const UPDATE_CARD_FREQUENCY: &str = "UPDATE_CARD_FREQUENCY";
const INACTIVE_CARD_PERIOD: &str = "INACTIVE_CARD_PERIOD";

pub fn load_ranker_config() -> Result<RankerConfig, String> {
    let mut config = RankerConfig {
        run_frequency: "".to_owned(),
        period_elapsed: "".to_owned(),
    };

    let run_frequency = env::var(UPDATE_CARD_FREQUENCY);
    if run_frequency.is_err() {
        return Err(format!(
            "missing environment variable: {}",
            UPDATE_CARD_FREQUENCY
        ));
    }

    config.run_frequency = run_frequency.unwrap();

    let period_elapsed = env::var(INACTIVE_CARD_PERIOD);
    if period_elapsed.is_err() {
        return Err(format!(
            "missing environment variable: {}",
            INACTIVE_CARD_PERIOD
        ));
    }

    config.period_elapsed = period_elapsed.unwrap();

    Ok(config)
}
