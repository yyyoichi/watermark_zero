import numpy as np
import json
import cv2

def create_dct_test_cases():
    """Create DCT test cases and output as JSON"""
    test_cases = []
    
    # Test case 1: Simple 2x2 data
    data1 = np.array([
        [1.0, 2.0],
        [3.0, 4.0]
    ], dtype=np.float32)
    
    dct1 = cv2.dct(data1)
    
    test_cases.append({
        "name": "2x2_simple",
        "input": {
            "data": data1.flatten().tolist(),
            "width": 2,
            "height": 2
        },
        "expected": {
            "dct": dct1.flatten().tolist()
        }
    })
    
    # Test case 2: 4x4 sequential data
    data2 = np.arange(1, 17, dtype=np.float32).reshape(4, 4)
    
    dct2 = cv2.dct(data2)
    
    test_cases.append({
        "name": "4x4_sequential",
        "input": {
            "data": data2.flatten().tolist(),
            "width": 4,
            "height": 4
        },
        "expected": {
            "dct": dct2.flatten().tolist()
        }
    })
    
    # Test case 3: 3x3 data (non-power-of-two)
    data3 = np.array([
        [1.0, 4.0, 7.0],
        [2.0, 5.0, 8.0],
        [3.0, 6.0, 9.0]
    ], dtype=np.float32)
    
    dct3 = cv2.dct(data3)
    
    test_cases.append({
        "name": "3x3_non_power_of_two",
        "input": {
            "data": data3.flatten().tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "dct": dct3.flatten().tolist()
        }
    })
    
    # Test case 4: 4x2 rectangular data
    data4 = np.array([
        [1.0, 2.0],
        [3.0, 4.0],
        [5.0, 6.0],
        [7.0, 8.0]
    ], dtype=np.float32)
    
    dct4 = cv2.dct(data4)
    
    test_cases.append({
        "name": "4x2_rectangular",
        "input": {
            "data": data4.flatten().tolist(),
            "width": 2,
            "height": 4
        },
        "expected": {
            "dct": dct4.flatten().tolist()
        }
    })
    
    # Test case 5: 2x4 rectangular data (different aspect ratio)
    data5 = np.array([
        [1.0, 2.0, 3.0, 4.0],
        [5.0, 6.0, 7.0, 8.0]
    ], dtype=np.float32)
    
    dct5 = cv2.dct(data5)
    
    test_cases.append({
        "name": "2x4_rectangular",
        "input": {
            "data": data5.flatten().tolist(),
            "width": 4,
            "height": 2
        },
        "expected": {
            "dct": dct5.flatten().tolist()
        }
    })
    
    # Test case 6: All zeros
    data6 = np.zeros((3, 3), dtype=np.float32)
    
    dct6 = cv2.dct(data6)
    
    test_cases.append({
        "name": "3x3_zeros",
        "input": {
            "data": data6.flatten().tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "dct": dct6.flatten().tolist()
        }
    })
    
    # Test case 7: Random data
    np.random.seed(42)  # for reproducible results
    data7 = np.random.rand(3, 4).astype(np.float32) * 10
    
    dct7 = cv2.dct(data7)
    
    test_cases.append({
        "name": "3x4_random",
        "input": {
            "data": data7.flatten().tolist(),
            "width": 4,
            "height": 3
        },
        "expected": {
            "dct": dct7.flatten().tolist()
        }
    })
    
    return test_cases

def main():
    print("Creating DCT test cases...")
    
    test_cases = create_dct_test_cases()
    
    # Output to JSON file
    with open('../test/dct_test_cases.json', 'w') as f:
        json.dump(test_cases, f, indent=2)
    
    print(f"Generated {len(test_cases)} test cases in dct_test_cases.json")
    
    # Display results
    for i, case in enumerate(test_cases):
        print(f"\nTest case {i+1}: {case['name']}")
        print(f"Input shape: {case['input']['height']}x{case['input']['width']}")
        input_data = case['input']['data']
        dct_data = case['expected']['dct']
        
        # Display first few input and DCT values
        print("Input values (first 6):", input_data[:6])
        print("DCT values (first 6):", [f"{x:.6f}" for x in dct_data[:6]])

if __name__ == "__main__":
    main()