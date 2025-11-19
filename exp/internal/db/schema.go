package db

const schema = `
-- Images table
CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uri TEXT NOT NULL UNIQUE
);

-- Image sizes table
CREATE TABLE IF NOT EXISTS image_sizes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_id INTEGER NOT NULL,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    UNIQUE(image_id, width, height)
);

-- Marks table (original watermark)
CREATE TABLE IF NOT EXISTS marks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mark BLOB NOT NULL,
    size INTEGER NOT NULL,
    UNIQUE(mark, size)
);

-- ECC marks table (encoded watermark)
CREATE TABLE IF NOT EXISTS ecc_marks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mark_id INTEGER NOT NULL,
    encoded BLOB NOT NULL,
    size INTEGER NOT NULL,
    algo_name TEXT NOT NULL,
    FOREIGN KEY (mark_id) REFERENCES marks(id) ON DELETE CASCADE,
    UNIQUE(mark_id, algo_name)
);

-- Mark parameters table (watermarking algorithm parameters)
CREATE TABLE IF NOT EXISTS mark_params (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    block_shape_h INTEGER NOT NULL,
    block_shape_w INTEGER NOT NULL,
    d1 INTEGER NOT NULL,
    d2 INTEGER NOT NULL,
    UNIQUE(block_shape_h, block_shape_w, d1, d2)
);

-- Results table
CREATE TABLE IF NOT EXISTS results (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    image_size_id INTEGER NOT NULL,
    ecc_mark_id INTEGER NOT NULL,
    mark_param_id INTEGER NOT NULL,
    
    original_image_path TEXT NOT NULL,
    embed_image_path TEXT NOT NULL,
    
    embed_count REAL NOT NULL,
    total_blocks INTEGER NOT NULL,
    
    encoded_accuracy REAL NOT NULL,
    decoded_accuracy REAL NOT NULL,
    success BOOLEAN NOT NULL,
    ssim REAL,
    
    FOREIGN KEY (image_size_id) REFERENCES image_sizes(id) ON DELETE CASCADE,
    FOREIGN KEY (ecc_mark_id) REFERENCES ecc_marks(id) ON DELETE CASCADE,
    FOREIGN KEY (mark_param_id) REFERENCES mark_params(id) ON DELETE CASCADE,
    UNIQUE(image_size_id, ecc_mark_id, mark_param_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_results_success ON results(success);
CREATE INDEX IF NOT EXISTS idx_results_embed_count ON results(embed_count);
CREATE INDEX IF NOT EXISTS idx_results_ssim ON results(ssim);
CREATE INDEX IF NOT EXISTS idx_results_accuracy ON results(decoded_accuracy);
CREATE INDEX IF NOT EXISTS idx_image_sizes_image ON image_sizes(image_id);
CREATE INDEX IF NOT EXISTS idx_image_sizes_dims ON image_sizes(width, height);
CREATE INDEX IF NOT EXISTS idx_ecc_marks_mark ON ecc_marks(mark_id);
CREATE INDEX IF NOT EXISTS idx_mark_params_d1d2 ON mark_params(d1, d2);

-- View for easy querying with all details
CREATE VIEW IF NOT EXISTS results_detailed AS
SELECT 
    r.id,
    
    i.uri as image_uri,
    isz.width,
    isz.height,
    
    mp.block_shape_h,
    mp.block_shape_w,
    mp.d1,
    mp.d2,
    
    em.algo_name as ecc_algo,
    em.size as encoded_size,
    m.size as original_size,
    
    r.embed_count,
    r.total_blocks,
    r.encoded_accuracy,
    r.decoded_accuracy,
    r.success,
    r.ssim,
    r.original_image_path,
    r.embed_image_path
FROM results r
JOIN image_sizes isz ON r.image_size_id = isz.id
JOIN images i ON isz.image_id = i.id
JOIN ecc_marks em ON r.ecc_mark_id = em.id
JOIN marks m ON em.mark_id = m.id
JOIN mark_params mp ON r.mark_param_id = mp.id;
`
