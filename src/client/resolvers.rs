use std::sync::atomic::AtomicUsize;
use std::sync::atomic::Ordering::Relaxed;

pub struct Resolvers {
    resolvers: Vec<String>,
    current_index: AtomicUsize,
}

impl Resolvers {
    pub fn new(resolvers: Vec<String>) -> Self {
        Self{
            resolvers,
            current_index: AtomicUsize::new(0),
        }
    }

    pub fn get_url(&self) -> &String {
        let prev_index = self.current_index.fetch_add(1, Relaxed);
        if prev_index == self.resolvers.len() - 1 {
            self.current_index.store(0, Relaxed);
        }
        &self.resolvers[prev_index]
    }
}