use crate::room::Room;

type ReceiveFunc = dyn Fn(u16, &[u8]) -> Result<(), anyhow::Error>;

trait RelayServiceTrait {
    fn receive_func(&self, stream_name: String) -> Result<&ReceiveFunc, anyhow::Error>;
    fn on_register_client(&self, client_id: u16) -> Result<(), anyhow::Error>;
    fn on_unregister_client(&self, client_id: u16) -> Result<(), anyhow::Error>;
    fn on_change_host(&self, client_id: u16) -> Result<(), anyhow::Error>;
    fn destroy(&self) -> Result<(), anyhow::Error>;
}

pub struct RelayService {
    room: Room,
}

impl RelayService {
    pub fn new(room: Room) -> Self {
        RelayService { room }
    }
}

impl RelayServiceTrait for RelayService {
    fn receive_func(&self, stream_name: String) -> Result<&ReceiveFunc, anyhow::Error> {
        anyhow::bail!("Not implemented");
    }
    fn on_register_client(&self, client_id: u16) -> Result<(), anyhow::Error> {
        anyhow::bail!("Not implemented");
    }
    fn on_unregister_client(&self, client_id: u16) -> Result<(), anyhow::Error> {
        anyhow::bail!("Not implemented");
    }
    fn on_change_host(&self, client_id: u16) -> Result<(), anyhow::Error> {
        anyhow::bail!("Not implemented");
    }
    fn destroy(&self) -> Result<(), anyhow::Error> {
        anyhow::bail!("Not implemented");
    }
}
