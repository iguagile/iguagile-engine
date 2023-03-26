use std::net::UdpSocket;
use std::sync::Arc;

pub trait ClientTrait {
    fn get_id(&self) -> u16;
    fn get_id_byte(&self) -> &[u8];
    fn read_loop(&mut self);
}

struct Client {
    id: u16,
    id_byte: [u8; 2],
    sock: Arc<UdpSocket>,
}

impl Client {
    pub fn new(id: u16, addr: String) -> Result<Self, anyhow::Error> {
        let sock = UdpSocket::bind(addr);
        if let Err(e) = sock {
            return Err(anyhow::anyhow!(e));
        }
        let id_byte = id.to_be_bytes();
        let sock = Arc::new(sock.unwrap());
        Ok(Client { id, id_byte, sock })
    }
}

impl ClientTrait for Client {
    fn get_id(&self) -> u16 {
        self.id
    }

    fn get_id_byte(&self) -> &[u8] {
        &self.id_byte
    }

    fn read_loop(&mut self) {
        let mut buf = [0; 1024];
        loop {
            let (amt, src) = self.sock.recv_from(&mut buf).unwrap();
            println!("Received {} bytes from {}", amt, src);
            self.sock.send_to(&buf, src).unwrap();
        }
    }
}
