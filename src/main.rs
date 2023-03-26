use crate::store::MemoryStore;
use std::process::exit;

mod client;
mod id;
mod room;

mod store;

fn main() {
    let redis = redis::Client::open("redis://127.0.0.1");
    if let Err(e) = redis {
        println!("redis error: {}", e);
        exit(1);
    }

    let m = MemoryStore::new(redis.unwrap());

    let _ = m;
    println!("Hello, world!");
}
