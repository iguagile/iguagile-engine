use crate::id::IdPool;
use crate::store::{iguagile, Store};
use std::{sync, time};

pub struct Engine<'a> {
    server_id: i64,
    // rooms: &'a sync::<i64, Room>,
    // factory: RoomServiceFactory,
    store: &'a dyn Store,
    id_pool: &'a dyn IdPool,
    server_proto: &'a iguagile::Server,
    room_update_duration: time::Duration,
    server_update_duration: time::Duration,
}

impl Engine<'_> {
    pub fn new(
        server_id: i64,
        store: &dyn Store,
        id_pool: &dyn IdPool,
        server_proto: &iguagile::Server,
        room_update_duration: time::Duration,
        server_update_duration: time::Duration,
    ) -> Self {
        Engine {
            server_id,
            store,
            id_pool,
            server_proto,
            room_update_duration,
            server_update_duration,
        }
    }
}
