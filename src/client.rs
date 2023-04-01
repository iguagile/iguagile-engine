use std::net::{ToSocketAddrs, UdpSocket};
use std::sync::Arc;

pub trait Client {
    fn get_id(&self) -> u16;
    fn read_loop(&mut self);
}

pub struct QUICClient {
    id: u16,
    sock: Arc<UdpSocket>,
}

impl QUICClient {
    pub fn new(id: u16, addr: impl ToSocketAddrs) -> Result<Self, anyhow::Error> {
        let sock = UdpSocket::bind(addr)?;
        let sock = Arc::new(sock);
        Ok(QUICClient { id, sock })
    }
}

impl Client for QUICClient {
    fn get_id(&self) -> u16 {
        self.id
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
