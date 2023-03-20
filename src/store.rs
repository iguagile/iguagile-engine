use std::io;

use iguagile::{Room, Server};

pub mod iguagile {
    tonic::include_proto!("iguagile");
}

trait Store {
    fn generate_server_id(&self) -> Result<i64, io::Error>;
    fn register_server(&self, s: Server) -> Result<(), io::Error>;
    fn unregister_server(&self, s: Server) -> Result<(), io::Error>;
    fn register_room(&self, r: Room) -> Result<(), io::Error>;
    fn unregister_room(&self, r: Room) -> Result<(), io::Error>;
}

pub struct MemoryStore {
    redis: redis::Client,
}

impl MemoryStore {
    pub fn new(c: redis::Client) -> Self {
        MemoryStore { redis: c }
    }
}

impl Store for MemoryStore {
    fn generate_server_id(&self) -> Result<i64, io::Error> {
        let conn = self.redis.get_connection();
        return if conn.is_err() {
            Err(io::Error::new(
                io::ErrorKind::Other,
                "redis connection error",
            ))
        } else {
            let mut conn = conn.unwrap();
            let result = redis::cmd("incr").arg("server_id").query::<i64>(&mut conn);
            if result.is_err() {
                Err(io::Error::new(io::ErrorKind::Other, "redis incr error"))
            } else {
                Ok(result.unwrap())
            }
        };
    }
    fn register_server(&self, s: Server) -> Result<(), io::Error> {
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
        let client = redis::Client::open("redis://127.0.0.1/").unwrap();
        let mut con = client.get_connection().unwrap();

        redis::cmd("del")
            .arg("server_id")
            .query::<i64>(&mut con)
            .unwrap();

        let m = MemoryStore::new(client);
        let id = m.generate_server_id().unwrap();
        assert_eq!(id, 1);
        let id = m.generate_server_id().unwrap();
        assert_eq!(id, 2);
    }
}
