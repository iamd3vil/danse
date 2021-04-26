use tokio::net::UdpSocket;
use std::io;
// use trust_dns_proto::op::message::Message;
use std::{net::SocketAddr, sync::Arc};

#[derive(Clone)]
struct Client {
    client: reqwest::Client,
}

impl Client {
    fn new() -> Self {
        Self {
            client: reqwest::Client::new()
        }
    }

    async fn process(&self, sock: &UdpSocket, buf: &[u8], addr: SocketAddr) {
        // let msg = Message::from_vec(buf).unwrap();
        let body = Vec::from(buf);
        let res = self.client.post("https://cloudflare-dns.com/dns-query")
            .body(body)
            .header("content-type", "application/dns-message")
            .header("content-type", "application/dns-message")
            .send().await
            .unwrap()
            .bytes().await
            .unwrap();
        sock.send_to(&res, addr).await.unwrap();
    }
}

#[tokio::main]
async fn main() -> io::Result<()> {
    let sock = UdpSocket::bind("127.0.0.1:5454").await?;
    let mut buf = [0; 4096];

    let cl = Client::new();
    let r = Arc::new(sock);

    loop {
        let (len, addr) = r.recv_from(&mut buf).await?;
        println!("{:?} bytes received", len);

        let c = cl.clone();
        let s = r.clone();

        tokio::spawn(async move {
            c.process(&s, &buf[..len], addr).await;
        });
    }
}
