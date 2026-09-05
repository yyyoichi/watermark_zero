use image::{DynamicImage, GenericImageView, RgbaImage, Rgba};

pub struct Yuv444Image {
    pub width: u32,
    pub height: u32,
    pub y: Vec<f32>,
    pub u: Vec<f32>,
    pub v: Vec<f32>,
    pub alpha: Vec<u8>,
}

impl Yuv444Image {
    /// 1. DynamicImage (RGBA) を YUV 444 (f32) に変換
    pub fn from_dynamic_image(img: &DynamicImage) -> Self {
        let (width, height) = img.dimensions();
        let size = (width * height) as usize;
        
        let mut y = Vec::with_capacity(size);
        let mut u = Vec::with_capacity(size);
        let mut v = Vec::with_capacity(size);
        let mut alpha = Vec::with_capacity(size);

        // ここで変換ロジックを回す（GoのColorToYUVBatch相当）
        
        Self { width, height, y, u, v, alpha }
    }

    /// 2. YUV 444 (f32) を DynamicImage (RGBA8) に戻す
    pub fn to_dynamic_image(&self) -> DynamicImage {
        let mut rgba_img = RgbaImage::new(self.width, self.height);
        
        // ここで逆変換ロジックを回す（GoのYUVToRGBA64Batch相当）
        // ※精度は落として 8-bit RGBA に戻すのが一般的
        
        DynamicImage::ImageRgba8(rgba_img)
    }
}