use serde_json::json;

use iguagile::{Room, Server};

pub mod iguagile {
    tonic::include_proto!("iguagile");
}

const REGISTER_SERVER: &str = "server_register";
const UNREGISTER_SERVER: &str = "server_unregister";
const REGISTER_ROOM: &str = "room_register";
const UNREGISTER_ROOM: &str = "room_unregister";

pub trait Store {
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
        let con = self.redis.get_connection();
        if let Err(e) = con {
            return Err(anyhow::anyhow!(e));
        }
        let mut con = con.unwrap();

        let payload = json!({
            "host": s.host,
            "port": s.port,
            "server_id": s.server_id,
            "token" : s.token,
            "api_port": s.api_port,
        });
        let result = redis::cmd("publish")
            .arg(REGISTER_SERVER)
            .arg(payload.to_string())
            .query::<i64>(&mut con);
        if let Err(e) = result {
            return Err(anyhow::anyhow!(e));
        }
        Ok(())
    }
    fn unregister_server(&self, s: Server) -> Result<(), anyhow::Error> {
        let con = self.redis.get_connection();
        if let Err(e) = con {
            return Err(anyhow::anyhow!(e));
        }
        let mut con = con.unwrap();

        let payload = json!({
            "host": s.host,
            "port": s.port,
            "server_id": s.server_id,
            "token" : s.token,
            "api_port": s.api_port,
        });
        let result = redis::cmd("publish")
            .arg(UNREGISTER_SERVER)
            .arg(payload.to_string())
            .query::<i64>(&mut con);
        if let Err(e) = result {
            return Err(anyhow::anyhow!(e));
        }
        Ok(())
    }
    fn register_room(&self, r: Room) -> Result<(), anyhow::Error> {
        let con = self.redis.get_connection();
        if let Err(e) = con {
            return Err(anyhow::anyhow!(e));
        }
        let mut con = con.unwrap();

        if r.server.is_none() {
            return Err(anyhow::anyhow!("server is none"));
        }

        let server = r.server.unwrap();
        let payload = json!({
            "room_id": r.room_id,
            "require_password": r.require_password,
            "max_user": r.max_user,
            "connected_user": r.connected_user,
            "server": {
                "host": server.host,
                "port": server.port,
                "server_id": server.server_id,
                "token" : server.token,
                "api_port": server.api_port,
            },
            "application_name": r.application_name,
            "version": r.version,
            "information": r.information,
        });
        let result = redis::cmd("publish")
            .arg(REGISTER_ROOM)
            .arg(payload.to_string())
            .query::<i64>(&mut con);
        if let Err(e) = result {
            return Err(anyhow::anyhow!(e));
        }
        Ok(())
    }
    fn unregister_room(&self, r: Room) -> Result<(), anyhow::Error> {
        let con = self.redis.get_connection();
        if let Err(e) = con {
            return Err(anyhow::anyhow!(e));
        }
        let mut con = con.unwrap();

        if r.server.is_none() {
            return Err(anyhow::anyhow!("server is none"));
        }

        let server = r.server.unwrap();
        let payload = json!({
            "room_id": r.room_id,
            "require_password": r.require_password,
            "max_user": r.max_user,
            "connected_user": r.connected_user,
            "server": {
                "host": server.host,
                "port": server.port,
                "server_id": server.server_id,
                "token" : server.token,
                "api_port": server.api_port,
            },
            "application_name": r.application_name,
            "version": r.version,
            "information": r.information,
        });
        let result = redis::cmd("publish")
            .arg(UNREGISTER_ROOM)
            .arg(payload.to_string())
            .query::<i64>(&mut con);
        if let Err(e) = result {
            return Err(anyhow::anyhow!(e));
        }

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
