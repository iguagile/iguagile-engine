use std::collections::HashMap;
use std::rc::Rc;

use anyhow::{anyhow, Error};

use crate::client::{Client, QUICClient};
use crate::engine::Engine;
use crate::id::{IdPool, MemoryIdPool};
use crate::store::{iguagile, MemoryStore, Store};

pub struct RoomConfig {
    room_id: u16,
    application_name: String,
    version: String,
    password: String,
    max_user: u16,
    info: HashMap<String, String>,
    token: Vec<u8>,
}

trait RoomTrait {
    fn join(&mut self, c: Box<dyn Client>) -> Result<(), anyhow::Error>;
    fn leave(&mut self, c: Box<dyn Client>) -> Result<(), anyhow::Error>;
}

trait ClientManagerTrait {
    fn get(&self, client_id: u16) -> Result<&dyn Client, anyhow::Error>;
    fn add(&mut self, client_id: u16, c: &dyn Client) -> Result<(), anyhow::Error>;
    fn remove(&mut self, client_id: u16) -> Result<(), anyhow::Error>;
    fn exists(&self, client_id: u16) -> bool;
    fn count(&self) -> u16;
    fn clear(&mut self);
    fn first(&self) -> Result<&dyn Client, anyhow::Error>;
}

struct ClientManager {
    clients: Rc<HashMap<u16, Box<dyn Client>>>,
}

impl ClientManager {
    pub fn new() -> Self {
        ClientManager {
            clients: Rc::new(HashMap::new()),
        }
    }
}

impl ClientManagerTrait for ClientManager {
    fn get(&self, client_id: u16) -> Result<&dyn Client, Error> {
        todo!()
    }

    fn add(&mut self, client_id: u16, c: &dyn Client) -> Result<(), Error> {
        todo!()
    }

    fn remove(&mut self, client_id: u16) -> Result<(), Error> {
        todo!()
    }

    fn exists(&self, client_id: u16) -> bool {
        todo!()
    }

    fn count(&self) -> u16 {
        todo!()
    }

    fn clear(&mut self) {
        todo!()
    }

    fn first(&self) -> Result<&dyn Client, Error> {
        todo!()
    }
}

pub struct Room {
    clients: Box<dyn ClientManagerTrait>,
    id_pool: Box<dyn IdPool>,
    host: Box<dyn Client>,
    config: RoomConfig,
    creator_connected: bool,
    room_proto: iguagile::Room,
    store: Box<dyn Store>,
    engine: Engine,
    // service: RoomService,
    streams: HashMap<String, Rc<dyn Client>>,
    ready: bool,
}

impl Room {
    pub fn new(config: RoomConfig) -> Self {
        Room {
            clients: Box::new(ClientManager::new()),
            id_pool: Box::new(MemoryIdPool::new()),
            host: Box::new(QUICClient::new(1, "127.0.0.1").unwrap()),
            config: config,
            creator_connected: false,
            room_proto: iguagile::Room::default(),
            store: Box::new(MemoryStore::new(redis::Client::open("redis://").unwrap())),
            engine: Engine::new(
                1,
                Box::new(MemoryStore::new(redis::Client::open("redis://").unwrap())),
                Box::new(MemoryIdPool::new()),
                &iguagile::Server::default(),
                std::time::Duration::from_secs(1),
                std::time::Duration::from_secs(1),
            ),
            streams: HashMap::new(),
            ready: false,
        }
    }
}

impl RoomTrait for Room {
    fn join(&mut self, c: Box<dyn Client>) -> Result<(), anyhow::Error> {
        let id = c.get_id();
        if self.clients.exists(id) {
            return Err(anyhow::anyhow!("client already joined"));
        }
        self.clients.add(id, &*c)
    }

    fn leave(&mut self, c: Box<dyn Client>) -> Result<(), anyhow::Error> {
        let id = c.get_id();
        if !self.clients.exists(id) {
            return Err(anyhow::anyhow!("client not joined"));
        }
        self.clients.remove(id)
    }
}
