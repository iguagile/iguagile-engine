use std::io;
use std::sync::atomic;

trait Store {
    fn generate_server_id(&self) -> Result<i64, io::Error>;
    fn register_server(&self) -> Result<(), io::Error>;
    fn unregister_server(&self) -> Result<(), io::Error>;
    fn register_room(&self) -> Result<(), io::Error>;
    fn unregister_room(&self)-> Result<(), io::Error>;
    fn new() -> MemoryStore;
}

pub struct MemoryStore {
    server_id : atomic::AtomicI64,

}

impl Store for MemoryStore {
    fn generate_server_id(&self) -> Result<i64, io::Error> {
        Ok(self.server_id.fetch_add(1, atomic::Ordering::SeqCst))
    }
    fn register_server(&self) -> Result<(), io::Error> {
        Ok(())
    }
    fn unregister_server(&self) -> Result<(), io::Error> {
        Ok(())
    }
    fn register_room(&self) -> Result<(), io::Error> {
        Ok(())
    }
    fn unregister_room(&self)-> Result<(), io::Error> {
        Ok(())
    }


    fn new() -> MemoryStore {
        MemoryStore {
            server_id : atomic::AtomicI64::new(0),
        }
    }
}