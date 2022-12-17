use std::{env, fmt};

pub struct DBConfig {
	pub user:     String,
	pub password: String,
	pub name:     String,
	pub host:     String,
	pub port:     String,
}

impl fmt::Display for DBConfig {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {

        write!(f, "connect_timeout=2 host={} user={} pasword={} dbname={}", self.host, self.user, self.password, self.name)
    }
}

pub struct GRPCServerConfig {
	pub host:              String,
	pub port:              String,
}

impl fmt::Display for GRPCServerConfig {
    fn fmt(&self, f: &mut fmt::Formatter<'_>) -> fmt::Result {
        //TODO Investigate how to use hostname instead
        write!(f, "0.0.0.0:{}",self.port) // self.host, 
    }
}

const DB_HOST: &str = "DB_HOST";
const DB_PORT: &str = "DB_PORT";
const DB_NAME: &str = "DB_NAME";
const DB_USER: &str = "DB_USER";
const DB_PASSWORD: &str = "DB_PASSWORD";

pub fn load_db_config() -> Result<DBConfig, String> {
    let mut config = DBConfig{ user: "".to_owned(), password: "".to_owned(), name: "".to_owned(), host: "".to_owned(), port: "".to_owned() };

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

    config.host = password.unwrap();

    Ok(config)
}

const GRPC_SERVER_HOST: &str = "GRPC_SERVER_HOST";
const GRPC_SERVER_PORT: &str = "GRPC_SERVER_PORT";

pub fn load_grpc_server_config() -> Result<GRPCServerConfig, String> {
    let mut config = GRPCServerConfig{ host: "".to_owned(), port: "".to_owned()};

    let host = env::var(GRPC_SERVER_HOST);
    if host.is_err() {
        return Err(format!("missing environment variable: {}", GRPC_SERVER_HOST));
    }

    config.host = host.unwrap();

    let port = env::var(GRPC_SERVER_PORT);
    if port.is_err() {
        return Err(format!("missing environment variable: {}", GRPC_SERVER_PORT));
    }

    config.port = port.unwrap();

    Ok(config)
}