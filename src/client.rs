use std::net::SocketAddr;
use trust_dns_proto::{op::{message::Message, Query}, serialize::binary::BinEncodable};
use tokio::net::UdpSocket;
use bytes::Bytes;
use reqwest::Error;
use std::time::Duration;
use lru::LruCache;
use tokio::sync::Mutex;
use chrono::prelude::*;
use base64::encode;
use std::sync::atomic::AtomicUsize;
use std::sync::atomic::Ordering::Relaxed;

const DEFAULT_RESOLVER: &str = "https://dns.quad9.net/dns-query";
const DEFAULT_CACHE: bool = true;

pub struct Client {
    client: reqwest::Client,
    settings: config::Config,
    resolvers: Resolvers,
    cache: Mutex<LruCache<String, CachedMsg>>
}

struct Resolvers {
    resolvers: Vec<String>,
    current_index: AtomicUsize,
}

impl Resolvers {
    fn new(resolvers: Vec<String>) -> Self {
        Self{
            resolvers,
            current_index: AtomicUsize::new(0),
        }
    }

    fn get_url(&self) -> &String {
        let prev_index = self.current_index.fetch_add(1, Relaxed);
        if prev_index == self.resolvers.len() - 1 {
            self.current_index.store(0, Relaxed);
        }
        &self.resolvers[prev_index]
    }
}


#[derive(Clone, Debug)]
struct CachedMsg {
    msg: Message,
    timestamp: DateTime<Utc>,
}

impl Client {
    pub fn new(settings: config::Config) -> Self {
        let urls =
            match settings.get_array("resolver.urls") {
                Ok(vals) => {
                    vals.iter()
                        .map(|val| val.to_string())
                        .collect()
                },
                Err(_) => vec![DEFAULT_RESOLVER.to_string()]
            };
        
        let resolvers = Resolvers::new(urls);

        Self {
            client: reqwest::Client::builder()
                .use_rustls_tls()
                .trust_dns(true)
                .timeout(Duration::new(60, 0))
                .build()
                .unwrap(),
            cache: Mutex::new(LruCache::new(100)),
            resolvers,
            settings,
        }
    }

    fn get_url(&self) -> &String {
        self.resolvers.get_url()
    }

    pub async fn process(&self, sock: &UdpSocket, buf: &[u8], addr: SocketAddr) {
        let shd_cache = match self.settings.get_bool("cache") {
            Ok(cache) => cache,
            Err(_) => DEFAULT_CACHE
        };
        if !shd_cache {
            let body = Vec::from(buf);
            let url = self.get_url();
            match get_response(&self.client, &url, body).await {
                Ok(res) => {
                    sock.send_to(&res, addr).await.unwrap();
                },
                Err(err) => println!("error sending request to resolver: {}", err)
            }
            return
        }
        // Check cache if there is a response already.
        let msg = Message::from_vec(buf).unwrap();
        let data: Option<Vec<u8>> = match self.get_from_cache(msg.queries()).await {
            Some(cached_msg) => {
                // Check if the ttl is expired
                let mut expired: bool = false;
                for ans in cached_msg.msg.answers() {
                    let diff = Utc::now() - cached_msg.timestamp;
                    if diff.num_nanoseconds().unwrap() > ans.ttl() as i64 * 1_000_000_000 {
                        expired = true;
                        break
                    }
                }
                // If expired, pop the cache
                if expired {
                    self.pop_from_cache(cached_msg.msg.queries()).await;
                    None
                } else {
                    // Change id and adjust ttl before sending.
                    let respmsg = adjust_resp(cached_msg.msg, msg.id(), cached_msg.timestamp).await;
                    Some(respmsg.to_bytes().unwrap())
                }
            },
            None => None
        };

        match data {
            Some(d) => {
                sock.send_to(&d, addr).await.unwrap();
            }
            None => {
                let body = Vec::from(buf);
                let url = self.get_url();
                match get_response(&self.client, &url, body).await {
                    Ok(res) => {
                        let ans = Message::from_vec(&res[..]).unwrap();
                        self.put_in_cache(msg.queries(), ans).await;
                        sock.send_to(&res, addr).await.unwrap();
                    },
                    Err(err) => println!("error sending request to resolver: {}", err)
                }
            }
        }
    }

    async fn get_from_cache(&self, queries: &[Query]) -> Option<CachedMsg> {
        // Make a string key out of queries
        self.cache.lock().await.get(&get_key(queries)).map(|b| b.to_owned())
    }

    async fn pop_from_cache(&self, queries: &[Query]) {
        self.cache.lock().await.pop(&get_key(queries));
    }

    async fn put_in_cache(&self, queries: &[Query], msg: Message) {
        let cached_msg = CachedMsg{
            msg,
            timestamp: Utc::now()
        };
        self.cache.lock().await.put(get_key(queries), cached_msg);
    }
}

async fn get_response(client: &reqwest::Client, url: &str, req: Vec<u8>) -> Result<Bytes, Error> {
    let encoded = encode(&req);
    let res = client.post(url)
        .body(req)
        .header("content-type", "application/dns-message")
        .header("content-type", "application/dns-message")
        .send().await;

    match res {
        Ok(res) => res.bytes().await,
        Err(e) if e.status().unwrap() == reqwest::StatusCode::METHOD_NOT_ALLOWED => {
            let res = client.get(url)
                .query(&[("dns", encoded)])
                .header("content-type", "application/dns-message")
                .send().await?.bytes().await?;
            Ok(res)
        }
        Err(e) => Err(e)
    }
}

async fn adjust_resp(msg: Message, id: u16,timestamp: DateTime<Utc>) -> Message {
    let mut respmsg = msg;
    respmsg.set_id(id);
    respmsg.answers_mut().iter_mut().for_each(|ans| {
        let expires_at = timestamp + chrono::Duration::seconds(ans.ttl() as i64);
        let ttl = (expires_at - Utc::now()).num_seconds();
        ans.set_ttl(ttl as u32);
    });
    respmsg
}

fn get_key(queries: &[Query]) -> String {
    let mut key = String::from("");
    for q in queries {
        key.push_str(&q.to_string());
        key.push_str("/n");
    }
    key
}