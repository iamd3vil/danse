mod client;
use config::{Config, File};
use tokio::net::UdpSocket;
use std::io;
use std::sync::Arc;

#[tokio::main]
async fn main() -> io::Result<()> {
    let settings = get_config("config.toml");    

    let bind_address = match settings.get_str("bind_address") {
        Ok(addr) => addr,
        Err(_) => "127.0.0.1:53".to_string()
    };

    // Setup logger
    let level = match settings.get_str("log_level") {
        Ok(level) => level,
        Err(_) => "info".to_string()
    };
    setup_logger(&level).unwrap();

    println!("Danse 🕺 is starting at {}", bind_address);

    let sock = UdpSocket::bind(&bind_address).await?;
    let mut buf = [0; 4096];

    let cl = Arc::new(client::Client::new(settings));
    let r = Arc::new(sock);


    loop {
        let (len, addr) = r.recv_from(&mut buf).await?;

        let c = cl.clone();
        let s = r.clone();

        tokio::spawn(async move {
            c.process(&s, &buf[..len], addr).await;
        });
    }
}

fn get_config(path: &str) -> config::Config {
    let mut settings = Config::default();
    settings.merge(File::with_name(path)).expect("error while reading config.toml");
    settings
}

fn setup_logger(level: &str) -> Result<(), fern::InitError> {
    let lvl = match level {
        "debug" => log::LevelFilter::Debug,
        _ => log::LevelFilter::Info,
    };
    fern::Dispatch::new()
        .format(|out, message, record| {
            out.finish(format_args!(
                "{}[{}][{}] {}",
                chrono::Local::now().format("[%Y-%m-%d][%H:%M:%S]"),
                record.target(),
                record.level(),
                message
            ))
        })
        .level(lvl)
        .chain(std::io::stdout())
        .apply()?;
    Ok(())
}