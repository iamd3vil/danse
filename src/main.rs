mod client;
use tokio::net::UdpSocket;
use std::io;
use std::sync::Arc;

#[tokio::main]
async fn main() -> io::Result<()> {
    let sock = UdpSocket::bind("127.0.0.1:5454").await?;
    let mut buf = [0; 4096];

    let cl = client::Client::new();
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
