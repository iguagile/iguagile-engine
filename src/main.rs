use crate::store::MemoryStore;

mod id_generator;
mod store;

fn main() {
    let redis = redis::Client::open("redis://127.0.0.1");
    if let Err(e) = redis {
        println!("redis error: {}", e);
        return;
    }
    let m = MemoryStore::new(redis.unwrap());
    let _ = m;
    println!("Hello, world!");
}
