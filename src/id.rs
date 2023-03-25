use std::collections::HashSet;

#[derive(Clone, Debug)]
pub struct IdPool {
    available_ids: HashSet<u16>,
}

impl IdPool {
    pub fn new() -> Self {
        let available_ids: HashSet<u16> = (0..=u16::MAX).collect();
        Self { available_ids }
    }

    pub fn is_empty(&self) -> bool {
        self.available_ids.is_empty()
    }

    pub fn len(&self) -> usize {
        self.available_ids.len()
    }

    pub fn clear(&mut self) {
        self.available_ids.clear();
        for i in 0..=u16::MAX {
            self.available_ids.insert(i);
        }
    }

    pub fn get_id(&mut self) -> Option<u16> {
        let id = self.available_ids.iter().next().cloned();
        if let Some(id) = id {
            self.available_ids.remove(&id);
        }
        id
    }

    pub fn return_id(&mut self, id: u16) -> bool {
        self.available_ids.insert(id)
    }
}

#[cfg(test)]
pub mod tests {
    use super::*;

    #[test]
    fn id_pool_new() {
        let pool = IdPool::new();
        assert_eq!(pool.available_ids.len(), 65536);
        for i in 0..=65535 {
            assert!(pool.available_ids.contains(&i));
        }
    }

    #[test]
    fn id_pool_is_empty() {
        let pool = IdPool {
            available_ids: [0].into(),
        };
        assert!(!pool.is_empty());
        let pool = IdPool {
            available_ids: Default::default(),
        };
        assert!(pool.is_empty());
    }

    #[test]
    fn id_pool_len() {
        let mut pool = IdPool::new();
        assert_eq!(pool.len(), 65536);
        assert!(pool.get_id().is_some());
        assert_eq!(pool.len(), 65535);
        assert!(pool.get_id().is_some());
        assert_eq!(pool.len(), 65534);
        pool.available_ids.clear();
        assert_eq!(pool.len(), 0);
    }

    #[test]
    fn id_pool_clear() {
        let mut pool = IdPool {
            available_ids: Default::default(),
        };
        assert_eq!(pool.available_ids.len(), 0);
        pool.clear();
        assert_eq!(pool.available_ids.len(), 65536);
    }

    #[test]
    fn id_pool_get_id() {
        let mut pool = IdPool::new();
        for _ in 0..=u16::MAX {
            let id = pool.get_id().unwrap();
            assert!(!pool.available_ids.contains(&id));
        }
        assert!(pool.get_id().is_none());
    }

    #[test]
    fn id_pool_return_id() {
        let mut pool = IdPool {
            available_ids: Default::default(),
        };
        for id in 0..=u16::MAX {
            assert!(!pool.available_ids.contains(&id));
            assert!(pool.return_id(id));
            assert!(pool.available_ids.contains(&id));
        }
        for id in 0..=u16::MAX {
            assert!(pool.available_ids.contains(&id));
            assert!(!pool.return_id(id));
            assert!(pool.available_ids.contains(&id));
        }
    }
}
