use std::net::SocketAddr;
use trust_dns_proto::op::message::Message;
use tokio::net::UdpSocket;
use bytes::Bytes;
use reqwest::Error;
use std::time::Duration;

const DEFAULT_RESOLVER: &str = "https://dns.quad9.net/dns-query";

#[derive(Clone)]
pub struct Client {
    client: reqwest::Client,
    settings: config::Config,
}

impl Client {
    pub fn new(settings: config::Config) -> Self {
        Self {
            client: reqwest::Client::builder()
                .use_rustls_tls()
                .timeout(Duration::new(60, 0))
                .build()
                .unwrap(),
            settings
        }
    }

    pub async fn process(&self, sock: &UdpSocket, buf: &[u8], addr: SocketAddr) {
        let msg = Message::from_vec(buf).unwrap();
        for query in msg.queries() {
            println!("Query: {}", query);
        }
        let body = Vec::from(buf);
        let url = match self.settings.get_str("resolver.address") {
            Ok(addr) => addr,
            Err(_) => String::from(DEFAULT_RESOLVER)
        };
        match get_response(&self.client, &url, body).await {
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
        .send().await;

    match res {
        Ok(res) => res.bytes().await,
        // TODO(sarat): If POST is not allowed, fallback to `GET`.
        Err(e) if e.status().unwrap() == reqwest::StatusCode::METHOD_NOT_ALLOWED => {
            Err(e)
        }
        Err(e) => Err(e)
    }
}