use crate::store::MemoryStore;

mod store;

fn main() {
    let redis = redis::Client::open("redis://127.0.0.1").unwrap();
    let m = MemoryStore::new(redis);
    let _ = m;
    println!("Hello, world!");
}
