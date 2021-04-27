use std::net::SocketAddr;
use trust_dns_proto::op::message::Message;
use tokio::net::UdpSocket;
use bytes::Bytes;
use reqwest::Error;

const DEFAULT_RESOLVER: &str = "https://cloudflare-dns.com/dns-query";

#[derive(Clone)]
pub struct Client {
    client: reqwest::Client,
}

impl Client {
    pub fn new() -> Self {
        Self {
            client: reqwest::Client::builder().use_rustls_tls().build().unwrap(),
        }
    }

    pub async fn process(&self, sock: &UdpSocket, buf: &[u8], addr: SocketAddr) {
        let msg = Message::from_vec(buf).unwrap();
        for query in msg.queries() {
            println!("Query: {}", query);
        }
        let body = Vec::from(buf);
        match get_response(&self.client, DEFAULT_RESOLVER, body).await {
            Ok(res) => {
                sock.send_to(&res, addr).await.unwrap();
                ()
            },
            Err(err) => println!("error sending request to resolver: {}", err)
        }
    }
}

async fn get_response(client: &reqwest::Client, url: &str, req: Vec<u8>) -> Result<Bytes, Error> {
    let res = client.post(url)
        .body(req)
        .header("content-type", "application/dns-message")
        .header("content-type", "application/dns-message")
        .send().await?
        .bytes().await?;
    Ok(res)
}