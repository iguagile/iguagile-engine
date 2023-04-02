use std::rc::Rc;
use std::time;

use anyhow::anyhow;

use crate::id::IdPool;
use crate::store::{iguagile, Store};

pub struct Engine {
    server_id: i64,
    // rooms: Rc<Hash<i64, Room>>,
    // factory: RoomServiceFactory,
    store: Box<dyn Store>,
    id_pool: Box<dyn IdPool>,
    server_proto: iguagile::Server,
    room_update_duration: time::Duration,
    server_update_duration: time::Duration,
}

impl Engine {
    pub fn new(
        server_id: i64,
        store: Box<dyn Store>,
        id_pool: Box<dyn IdPool>,
        server_proto: &iguagile::Server,
        room_update_duration: time::Duration,
        server_update_duration: time::Duration,
    ) -> Self {
        Engine {
            server_id: server_id,
            // rooms: Rc::new(Hash::new()),
            // factory: RoomServiceFactory::new(),
            store: store,
            id_pool: id_pool,
            server_proto: server_proto.clone(),
            room_update_duration: room_update_duration,
            server_update_duration: server_update_duration,
        }
    }
}
