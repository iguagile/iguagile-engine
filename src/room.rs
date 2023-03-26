use crate::client::ClientTrait;
use std::collections::HashMap;

trait RoomTrait<'a> {
    fn join(&mut self, c: &'a dyn ClientTrait) -> Result<(), anyhow::Error>;
    fn leave(&mut self, c: &'a dyn ClientTrait) -> Result<(), anyhow::Error>;
}

struct Room<'a> {
    id: u16,
    clients: HashMap<u16, &'a dyn ClientTrait>,
}

impl<'a> Room<'a> {
    pub fn new(id: u16) -> Self {
        Room {
            id,
            clients: HashMap::new(),
        }
    }
}

impl<'a> RoomTrait<'a> for Room<'a> {
    fn join(&mut self, c: &'a dyn ClientTrait) -> Result<(), anyhow::Error> {
        let id = c.get_id();
        if self.clients.contains_key(&id) {
            return Err(anyhow::anyhow!("client already joined"));
        }
        self.clients.insert(id, c);
        Ok(())
    }

    fn leave(&mut self, c: &'a dyn ClientTrait) -> Result<(), anyhow::Error> {
        let id = c.get_id();
        if !self.clients.contains_key(&id) {
            return Err(anyhow::anyhow!("client not joined"));
        }
        self.clients.remove(&id);
        Ok(())
    }
}
