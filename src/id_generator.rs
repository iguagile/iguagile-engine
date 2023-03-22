trait IdGenerator {
    fn allocate(id: i64) -> Result<(), anyhow::Error>;
    fn free(id: i64) -> Result<(), anyhow::Error>;
    fn free_all() -> Result<(), anyhow::Error>;
    fn is_allocated(id: i64) -> Result<bool, anyhow::Error>;
    fn get_allocated_idcount() -> Result<i64, anyhow::Error>;
}

pub struct MemoryIdGenerator {
    map: bitmaps::Bitmap<1024>,
}

impl MemoryIdGenerator {
    pub fn new() -> Self {
        MemoryIdGenerator {
            map: bitmaps::Bitmap::new(),
        }
    }
}
