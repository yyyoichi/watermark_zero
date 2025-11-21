package db

const schema = `
-- Images table
CREATE TABLE IF NOT EXISTS images (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    uri TEXT NOT NULL UNIQUE
);

-- Image sizes table (independent of source image)
CREATE TABLE IF NOT EXISTS image_sizes (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    width INTEGER NOT NULL,
    height INTEGER NOT NULL,
    UNIQUE(width, height)
);

-- Marks table (original watermark)
CREATE TABLE IF NOT EXISTS marks (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    mark BLOB NOT NULL UNIQUE
);

-- Mark ECC algorithms table (independent of mark data)
CREATE TABLE IF NOT EXISTS mark_ecc_algos (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    algo_name TEXT NOT NULL UNIQUE
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
    image_id INTEGER NOT NULL,
    image_size_id INTEGER NOT NULL,
    mark_id INTEGER NOT NULL,
    mark_ecc_algo_id INTEGER NOT NULL,
    mark_param_id INTEGER NOT NULL,
    
    embed_count REAL NOT NULL,
    total_blocks INTEGER NOT NULL,
    
    encoded_accuracy REAL NOT NULL,
    decoded_accuracy REAL NOT NULL,
    success BOOLEAN NOT NULL,
    ssim REAL,
    
    FOREIGN KEY (image_id) REFERENCES images(id) ON DELETE CASCADE,
    FOREIGN KEY (image_size_id) REFERENCES image_sizes(id) ON DELETE CASCADE,
    FOREIGN KEY (mark_id) REFERENCES marks(id) ON DELETE CASCADE,
    FOREIGN KEY (mark_ecc_algo_id) REFERENCES mark_ecc_algos(id) ON DELETE CASCADE,
    FOREIGN KEY (mark_param_id) REFERENCES mark_params(id) ON DELETE CASCADE,
    UNIQUE(image_id, image_size_id, mark_id, mark_ecc_algo_id, mark_param_id)
);

-- Indexes for performance
CREATE INDEX IF NOT EXISTS idx_results_success ON results(success);
CREATE INDEX IF NOT EXISTS idx_results_embed_count ON results(embed_count);
CREATE INDEX IF NOT EXISTS idx_results_ssim ON results(ssim);
CREATE INDEX IF NOT EXISTS idx_results_accuracy ON results(decoded_accuracy);
CREATE INDEX IF NOT EXISTS idx_results_image ON results(image_id);
CREATE INDEX IF NOT EXISTS idx_image_sizes_dims ON image_sizes(width, height);
CREATE INDEX IF NOT EXISTS idx_mark_params_d1d2 ON mark_params(d1, d2);

-- View for easy querying with all details
CREATE VIEW IF NOT EXISTS results_view AS
SELECT 
    r.id,
    
    i.uri as image_uri,
    isz.width,
    isz.height,
    
    mp.block_shape_h,
    mp.block_shape_w,
    mp.d1,
    mp.d2,
    
    mea.algo_name as ecc_algo,
    
    r.embed_count,
    r.total_blocks,
    r.encoded_accuracy,
    r.decoded_accuracy,
    r.success,
    r.ssim
FROM results r
JOIN images i ON r.image_id = i.id
JOIN image_sizes isz ON r.image_size_id = isz.id
JOIN marks m ON r.mark_id = m.id
JOIN mark_ecc_algos mea ON r.mark_ecc_algo_id = mea.id
JOIN mark_params mp ON r.mark_param_id = mp.id;
`
