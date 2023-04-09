use std::collections::HashMap;
use std::net::{SocketAddr, UdpSocket};
use std::rc::Rc;
use std::sync::Arc;
use std::time;

use mio::{net, Events, Interest, Poll, Token};
use ring::rand::SystemRandom;

use crate::id::IdPool;
use crate::room::Room;
use crate::store::{iguagile, Store};

const MAX_DATAGRAM_SIZE: usize = 1350;

pub struct Engine {
    server_id: i64,
    rooms: Rc<HashMap<i64, Room>>,
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
        server_proto: iguagile::Server,
        room_update_duration: time::Duration,
        server_update_duration: time::Duration,
    ) -> Self {
        Engine {
            server_id: server_id,
            rooms: Rc::new(HashMap::new()),
            store: store,
            id_pool: id_pool,
            server_proto: server_proto,
            room_update_duration: room_update_duration,
            server_update_duration: server_update_duration,
        }
    }

    pub fn serve(&self, socket: Arc<UdpSocket>) -> Result<(), anyhow::Error> {
        // let mut room_update_timer = time::Instant::now();
        // let mut server_update_timer = time::Instant::now();
        // loop {
        //     if room_update_timer.elapsed() > self.room_update_duration {
        //         self.update_rooms()?;
        //         room_update_timer = time::Instant::now();
        //     }
        //     if server_update_timer.elapsed() > self.server_update_duration {
        //         self.update_server()?;
        //         server_update_timer = time::Instant::now();
        //     }
        // }
        todo!()

        // self.rooms.get(&self.server_id).unwrap().serve(socket)
    }

    pub fn start(
        self,
        serverAddress: SocketAddr,
        apiAddress: SocketAddr,
    ) -> Result<(), anyhow::Error> {
        let mut poll = Poll::new().unwrap();
        let mut socket = net::UdpSocket::bind(serverAddress).unwrap();

        poll.registry()
            .register(&mut socket, Token(0), Interest::READABLE)
            .unwrap();

        let mut config = quiche::Config::new(quiche::PROTOCOL_VERSION).unwrap();
        config
            .load_cert_chain_from_pem_file("examples/cert.crt")
            .unwrap();
        config
            .load_priv_key_from_pem_file("examples/cert.key")
            .unwrap();
        config
            .set_application_protos(&[b"hq-interop", b"hq-29", b"hq-28", b"hq-27", b"http/0.9"])
            .unwrap();

        config.set_max_idle_timeout(5000);
        config.set_max_recv_udp_payload_size(MAX_DATAGRAM_SIZE);
        config.set_max_send_udp_payload_size(MAX_DATAGRAM_SIZE);
        config.set_initial_max_data(10_000_000);
        config.set_initial_max_stream_data_bidi_local(1_000_000);
        config.set_initial_max_stream_data_bidi_remote(1_000_000);
        config.set_initial_max_stream_data_uni(1_000_000);
        config.set_initial_max_streams_bidi(100);
        config.set_initial_max_streams_uni(100);
        config.set_disable_active_migration(true);
        config.enable_early_data();

        let rng = SystemRandom::new();
        let conn_id_seed = ring::hmac::Key::generate(ring::hmac::HMAC_SHA256, &rng).unwrap();
        let mut events = Events::with_capacity(1024);
        let local_addr = socket.local_addr().unwrap();

        loop {
            // Find the shorter timeout from all the active connections.
            //
            // TODO: use event loop that properly supports timers
            // let timeout = clients.values().filter_map(|c| c.conn.timeout()).min()a

            let timeout = None;
            poll.poll(&mut events, timeout).unwrap();

            // Read incoming UDP packets from the socket and feed them to quiche,
            // until there are no more packets to read.
            'read: loop {}
        }
        return Ok(());
    }
}

// test
#[cfg(test)]
mod tests {}
