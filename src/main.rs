use crate::store::MemoryStore;

mod store;

fn main() {
    let m = MemoryStore::new();
    let _ = m;
    println!("Hello, world!");
}
