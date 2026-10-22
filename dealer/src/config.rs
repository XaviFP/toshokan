use std::{env, fmt};

use sqlx::postgres::PgConnectOptions;

pub struct DBConfig {
    pub user: String,
    pub password: String,
    pub name: String,
    pub host: String,
    pub port: String,
}

impl DBConfig {
    pub fn connect_options(&self) -> Result<PgConnectOptions, String> {
        let port: u16 = self
            .port
            .parse()
            .map_err(|_| format!("invalid {}: {}", DB_PORT, self.port))?;

        Ok(PgConnectOptions::new()
            .host(&self.host)
            .port(port)
            .username(&self.user)
            .password(&self.password)
            .database(&self.name))
    }
}

pub struct CourseServiceConfig {
    pub host: String,
    pub port: String,
}

impl fmt::Display for CourseServiceConfig {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        write!(f, "http://{}:{}", self.host, self.port)
    }
}

pub struct GRPCServerConfig {
    pub host: String,
    pub port: String,
}

impl fmt::Display for GRPCServerConfig {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        //TODO Investigate how to use hostname instead
        write!(f, "0.0.0.0:{}", self.port) // self.host,
    }
}

const DB_HOST: &str = "DB_HOST";
const DB_PORT: &str = "DB_PORT";
const DB_NAME: &str = "DB_NAME";
const DB_USER: &str = "DB_USER";
const DB_PASSWORD: &str = "DB_PASSWORD";

pub fn load_db_config() -> Result<DBConfig, String> {
    let mut config = DBConfig {
        user: "".to_owned(),
        password: "".to_owned(),
        name: "".to_owned(),
        host: "".to_owned(),
        port: "".to_owned(),
    };

    let host = env::var(DB_HOST);
    if host.is_err() {
        return Err(format!("missing environment variable: {}", DB_HOST));
    }

    config.host = host.unwrap();

    let port = env::var(DB_PORT);
    if port.is_err() {
        return Err(format!("missing environment variable: {}", DB_PORT));
    }

    config.port = port.unwrap();

    let name = env::var(DB_NAME);
    if name.is_err() {
        return Err(format!("missing environment variable: {}", DB_NAME));
    }

    config.name = name.unwrap();

    let user = env::var(DB_USER);
    if user.is_err() {
        return Err(format!("missing environment variable: {}", DB_USER));
    }

    config.user = user.unwrap();

    let password = env::var(DB_PASSWORD);
    if password.is_err() {
        return Err(format!("missing environment variable: {}", DB_PASSWORD));
    }

    config.password = password.unwrap();

    Ok(config)
}

const GRPC_SERVER_HOST: &str = "GRPC_SERVER_HOST";
const GRPC_SERVER_PORT: &str = "GRPC_SERVER_PORT";

pub fn load_grpc_server_config() -> Result<GRPCServerConfig, String> {
    let mut config = GRPCServerConfig {
        host: "".to_owned(),
        port: "".to_owned(),
    };

    let host = env::var(GRPC_SERVER_HOST);
    if host.is_err() {
        return Err(format!(
            "missing environment variable: {}",
            GRPC_SERVER_HOST
        ));
    }

    config.host = host.unwrap();

    let port = env::var(GRPC_SERVER_PORT);
    if port.is_err() {
        return Err(format!(
            "missing environment variable: {}",
            GRPC_SERVER_PORT
        ));
    }

    config.port = port.unwrap();

    Ok(config)
}

const COURSE_GRPC_SERVER_HOST: &str = "COURSE_GRPC_SERVER_HOST";
const COURSE_GRPC_SERVER_PORT: &str = "COURSE_GRPC_SERVER_PORT";

pub fn load_course_service_config() -> Result<CourseServiceConfig, String> {
    let mut config = CourseServiceConfig {
        host: "".to_owned(),
        port: "".to_owned(),
    };

    let host = env::var(COURSE_GRPC_SERVER_HOST);
    if host.is_err() {
        return Err(format!(
            "missing environment variable: {}",
            COURSE_GRPC_SERVER_HOST
        ));
    }

    config.host = host.unwrap();

    let port = env::var(COURSE_GRPC_SERVER_PORT);
    if port.is_err() {
        return Err(format!(
            "missing environment variable: {}",
            COURSE_GRPC_SERVER_PORT
        ));
    }

    config.port = port.unwrap();

    Ok(config)
}

#[cfg(test)]
mod tests {
    use super::DBConfig;

    #[test]
    fn connect_options_rejects_a_port_that_is_not_a_number() {
        let config = DBConfig {
            user: "toshokan".to_owned(),
            password: "secret".to_owned(),
            name: "deck".to_owned(),
            host: "localhost".to_owned(),
            port: "not-a-port".to_owned(),
        };

        let actual = config.connect_options().unwrap_err();

        assert_eq!("invalid DB_PORT: not-a-port", actual);
    }
}
