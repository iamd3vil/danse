mod client;
use config::{Config, File};
use tokio::net::UdpSocket;
use std::io;
use std::sync::Arc;

#[tokio::main]
async fn main() -> io::Result<()> {
    let mut settings = Config::default();
    settings.merge(File::with_name("config.toml")).expect("error while reading config.toml");

    let bind_address = match settings.get_str("bind_address") {
        Ok(addr) => addr,
        Err(_) => "127.0.0.1:53".to_string()
    };

    println!("bind address: {}", bind_address);

    let sock = UdpSocket::bind(&bind_address).await?;
    let mut buf = [0; 4096];

    let cl = client::Client::new(settings);
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
