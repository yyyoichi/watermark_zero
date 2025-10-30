import numpy as np
import json
import cv2

def create_yuv_test_cases():
    """Create YUV conversion test cases and output as JSON"""
    test_cases = []
    
    # Test case 1: Simple 2x2 BGR data (OpenCV uses BGR by default)
    bgr1 = np.array([
        [[0, 0, 255], [0, 255, 0]],      # Red, Green (in BGR)
        [[255, 0, 0], [255, 255, 255]]   # Blue, White (in BGR)
    ], dtype=np.float32)

    yuv1 = cv2.cvtColor(bgr1, cv2.COLOR_BGR2YUV)
    
    # Convert BGR to RGB for consistent input format
    rgb1 = cv2.cvtColor(bgr1, cv2.COLOR_BGR2RGB)
    
    test_cases.append({
        "name": "2x2_primary_colors",
        "input": {
            "rgb": rgb1.flatten().astype(np.uint8).tolist(),
            "width": 2,
            "height": 2
        },
        "expected": {
            "yuv": yuv1.flatten().tolist()
        }
    })
    
    # Test case 2: 3x3 grayscale gradient (BGR format)
    bgr2 = np.array([
        [[0, 0, 0], [64, 64, 64], [128, 128, 128]],
        [[192, 192, 192], [255, 255, 255], [32, 32, 32]],
        [[96, 96, 96], [160, 160, 160], [224, 224, 224]]
    ], dtype=np.float32)
    
    yuv2 = cv2.cvtColor(bgr2, cv2.COLOR_BGR2YUV)
    
    # Convert BGR to RGB for consistent input format
    rgb2 = cv2.cvtColor(bgr2, cv2.COLOR_BGR2RGB)
    
    test_cases.append({
        "name": "3x3_grayscale",
        "input": {
            "rgb": rgb2.flatten().astype(np.uint8).tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "yuv": yuv2.flatten().tolist()
        }
    })
    
    # Test case 3: 4x4 random colors
    np.random.seed(42)  # for reproducible results
    bgr3 = np.random.randint(0, 256, (4, 4, 3)).astype(np.float32)
    
    yuv3 = cv2.cvtColor(bgr3, cv2.COLOR_BGR2YUV)
    
    # Convert BGR to RGB for consistent input format
    rgb3 = cv2.cvtColor(bgr3, cv2.COLOR_BGR2RGB)
    
    test_cases.append({
        "name": "4x4_random",
        "input": {
            "rgb": rgb3.flatten().astype(np.uint8).tolist(),
            "width": 4,
            "height": 4
        },
        "expected": {
            "yuv": yuv3.flatten().tolist()
        }
    })
    
    # Test case 4: Edge cases (black, white, mid-gray)
    bgr4 = np.array([
        [[0, 0, 0], [255, 255, 255]],    # Black, White
        [[128, 128, 128], [127, 127, 127]]  # Mid-gray variations
    ], dtype=np.float32)
    
    yuv4 = cv2.cvtColor(bgr4, cv2.COLOR_BGR2YUV)
    
    # Convert BGR to RGB for consistent input format
    rgb4 = cv2.cvtColor(bgr4, cv2.COLOR_BGR2RGB)
    
    test_cases.append({
        "name": "2x2_edge_cases",
        "input": {
            "rgb": rgb4.flatten().astype(np.uint8).tolist(),
            "width": 2,
            "height": 2
        },
        "expected": {
            "yuv": yuv4.flatten().tolist()
        }
    })
    
    # Test case 5: Single pixel (for simple validation)
    bgr5 = np.array([[[200, 150, 100]]], dtype=np.float32)  # BGR format
    yuv5 = cv2.cvtColor(bgr5, cv2.COLOR_BGR2YUV)
    
    # Convert BGR to RGB for consistent input format  
    rgb5 = cv2.cvtColor(bgr5, cv2.COLOR_BGR2RGB)
    
    test_cases.append({
        "name": "1x1_single_pixel",
        "input": {
            "rgb": rgb5.flatten().astype(np.uint8).tolist(),
            "width": 1,
            "height": 1
        },
        "expected": {
            "yuv": yuv5.flatten().tolist()
        }
    })
    
    return test_cases

def main():
    print("Creating YUV conversion test cases...")
    
    test_cases = create_yuv_test_cases()
    
    # Output to JSON file
    with open('../test/yuv_test_cases.json', 'w') as f:
        json.dump(test_cases, f, indent=2)
    
    print(f"Generated {len(test_cases)} test cases in yuv_test_cases.json")
    
    # Display results
    for i, case in enumerate(test_cases):
        print(f"\nTest case {i+1}: {case['name']}")
        print(f"Input shape: {case['input']['height']}x{case['input']['width']}")
        rgb_data = case['input']['rgb']
        yuv_data = case['expected']['yuv']
        
        # Display first few RGB and YUV values
        print("RGB values (first 6):", rgb_data[:6])
        print("YUV values (first 6):", yuv_data[:6])
        
        # Show RGB->YUV conversion for first pixel
        if len(rgb_data) >= 3:
            r, g, b = rgb_data[0], rgb_data[1], rgb_data[2]
            y, u, v = yuv_data[0], yuv_data[1], yuv_data[2]
            print(f"First pixel: RGB({r}, {g}, {b}) -> YUV({y}, {u}, {v})")

if __name__ == "__main__":
    main()