use iguagile::{Room, Server};

pub mod iguagile {
    tonic::include_proto!("iguagile");
}

trait Store {
    fn generate_server_id(&self) -> Result<i64, anyhow::Error>;
    fn register_server(&self, s: Server) -> Result<(), anyhow::Error>;
    fn unregister_server(&self, s: Server) -> Result<(), anyhow::Error>;
    fn register_room(&self, r: Room) -> Result<(), anyhow::Error>;
    fn unregister_room(&self, r: Room) -> Result<(), anyhow::Error>;
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
    fn generate_server_id(&self) -> Result<i64, anyhow::Error> {
        let con = self.redis.get_connection();
        if let Err(e) = con {
            return Err(anyhow::anyhow!(e));
        }

        let id = redis::cmd("incr")
            .arg("server_id")
            .query::<i64>(&mut con.unwrap())
            .unwrap();
        Ok(id)
    }
    fn register_server(&self, s: Server) -> Result<(), anyhow::Error> {
        let _ = s;
        Ok(())
    }
    fn unregister_server(&self, s: Server) -> Result<(), anyhow::Error> {
        let _ = s;
        Ok(())
    }
    fn register_room(&self, r: Room) -> Result<(), anyhow::Error> {
        let _ = r;
        Ok(())
    }
    fn unregister_room(&self, r: Room) -> Result<(), anyhow::Error> {
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
