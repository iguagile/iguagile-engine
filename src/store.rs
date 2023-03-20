use std::io;
use std::sync::atomic;

use iguagile::{Server, Room};

pub mod iguagile {
    tonic::include_proto!("iguagile");
}

trait Store {
    fn generate_server_id(&self) -> Result<i64, io::Error>;
    fn register_server(&self, s : Server) -> Result<(), io::Error>;
    fn unregister_server(&self, s: Server) -> Result<(), io::Error>;
    fn register_room(&self, r: Room) -> Result<(), io::Error>;
    fn unregister_room(&self, r: Room) -> Result<(), io::Error>;
}

#[derive(Default)]
pub struct MemoryStore {
    server_id: atomic::AtomicI64,
}

impl MemoryStore {
    pub fn new() -> Self {
        MemoryStore {
            server_id: atomic::AtomicI64::new(0),
        }
    }
}

impl Store for MemoryStore {
    fn generate_server_id(&self) -> Result<i64, io::Error> {
        Ok(self.server_id.fetch_add(1, atomic::Ordering::SeqCst))
    }
    fn register_server(&self, s :Server) -> Result<(), io::Error> {
        let _ = s;
        Ok(())
    }
    fn unregister_server(&self, s: Server) -> Result<(), io::Error> {
        let _ = s;
        Ok(())
    }
    fn register_room(&self, r: Room) -> Result<(), io::Error> {
        let _ = r;

        Ok(())
    }
    fn unregister_room(&self, r: Room) -> Result<(), io::Error> {
        let _ = r;
        Ok(())
    }
}


#[cfg(test)]
mod tests {
    use super::*;

    #[test]
    fn test_generate_server_id() {
        let m = MemoryStore::new();
        let id = m.generate_server_id().unwrap();
        assert_eq!(id, 0);
        let id = m.generate_server_id().unwrap();
        assert_eq!(id, 1);
    }
}