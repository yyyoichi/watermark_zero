import numpy as np
import json
from pywt import dwt2

def create_test_cases():
    """Create DWT test cases and output as JSON"""
    test_cases = []
    
    # Test case 1: Simple 4x4 data
    data1 = np.array([
        [1.0, 2.0, 3.0, 4.0],
        [5.0, 6.0, 7.0, 8.0],
        [9.0, 10.0, 11.0, 12.0],
        [13.0, 14.0, 15.0, 16.0]
    ], dtype=np.float32)
    
    coeffs1 = dwt2(data1, 'haar')
    cA1, (cH1, cV1, cD1) = coeffs1
    
    test_cases.append({
        "name": "4x4_simple",
        "input": {
            "data": data1.flatten().tolist(),
            "width": 4,
            "height": 4
        },
        "expected": {
            "cA": cA1.flatten().tolist(),
            "cH": cH1.flatten().tolist(),
            "cV": cV1.flatten().tolist(),
            "cD": cD1.flatten().tolist()
        }
    })
    
    # Test case 2: 6x4 rectangular data
    data2 = np.array([
        [1.0, 3.0, 5.0, 7.0],
        [2.0, 4.0, 6.0, 8.0],
        [9.0, 11.0, 13.0, 15.0],
        [10.0, 12.0, 14.0, 16.0],
        [17.0, 19.0, 21.0, 23.0],
        [18.0, 20.0, 22.0, 24.0]
    ], dtype=np.float32)
    
    coeffs2 = dwt2(data2, 'haar')
    cA2, (cH2, cV2, cD2) = coeffs2
    
    test_cases.append({
        "name": "6x4_rectangle",
        "input": {
            "data": data2.flatten().tolist(),
            "width": 4,
            "height": 6
        },
        "expected": {
            "cA": cA2.flatten().tolist(),
            "cH": cH2.flatten().tolist(),
            "cV": cV2.flatten().tolist(),
            "cD": cD2.flatten().tolist()
        }
    })
    
    # Test case 3: 3x3 odd-sized data
    data3 = np.array([
        [1.0, 2.0, 3.0],
        [4.0, 5.0, 6.0],
        [7.0, 8.0, 9.0]
    ], dtype=np.float32)
    
    coeffs3 = dwt2(data3, 'haar')
    cA3, (cH3, cV3, cD3) = coeffs3
    
    test_cases.append({
        "name": "3x3_odd",
        "input": {
            "data": data3.flatten().tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "cA": cA3.flatten().tolist(),
            "cH": cH3.flatten().tolist(),
            "cV": cV3.flatten().tolist(),
            "cD": cD3.flatten().tolist()
        }
    })
    
    # Test case 4: 8x8 square data
    data4 = np.arange(1, 65, dtype=np.float32).reshape(8, 8)
    
    coeffs4 = dwt2(data4, 'haar')
    cA4, (cH4, cV4, cD4) = coeffs4
    
    test_cases.append({
        "name": "8x8_square",
        "input": {
            "data": data4.flatten().tolist(),
            "width": 8,
            "height": 8
        },
        "expected": {
            "cA": cA4.flatten().tolist(),
            "cH": cH4.flatten().tolist(),
            "cV": cV4.flatten().tolist(),
            "cD": cD4.flatten().tolist()
        }
    })
    
    # Test case 5: 16x8 rectangular data
    data5 = np.random.seed(42)  # fixed seed for reproducible results
    data5 = np.random.rand(16, 8).astype(np.float32) * 100
    
    coeffs5 = dwt2(data5, 'haar')
    cA5, (cH5, cV5, cD5) = coeffs5
    
    test_cases.append({
        "name": "16x8_random",
        "input": {
            "data": data5.flatten().tolist(),
            "width": 8,
            "height": 16
        },
        "expected": {
            "cA": cA5.flatten().tolist(),
            "cH": cH5.flatten().tolist(),
            "cV": cV5.flatten().tolist(),
            "cD": cD5.flatten().tolist()
        }
    })
    
    # Test case 6: 16x16 large square data
    # Checkerboard pattern
    data6 = np.zeros((16, 16), dtype=np.float32)
    for i in range(16):
        for j in range(16):
            if (i + j) % 2 == 0:
                data6[i, j] = 100.0
            else:
                data6[i, j] = 0.0
    
    coeffs6 = dwt2(data6, 'haar')
    cA6, (cH6, cV6, cD6) = coeffs6
    
    test_cases.append({
        "name": "16x16_checkerboard",
        "input": {
            "data": data6.flatten().tolist(),
            "width": 16,
            "height": 16
        },
        "expected": {
            "cA": cA6.flatten().tolist(),
            "cH": cH6.flatten().tolist(),
            "cV": cV6.flatten().tolist(),
            "cD": cD6.flatten().tolist()
        }
    })
    
    # Test case 7: 10x12 non-power-of-two size
    data7 = np.linspace(1, 120, 120, dtype=np.float32).reshape(10, 12)
    
    coeffs7 = dwt2(data7, 'haar')
    cA7, (cH7, cV7, cD7) = coeffs7
    
    test_cases.append({
        "name": "10x12_non_power_of_two",
        "input": {
            "data": data7.flatten().tolist(),
            "width": 12,
            "height": 10
        },
        "expected": {
            "cA": cA7.flatten().tolist(),
            "cH": cH7.flatten().tolist(),
            "cV": cV7.flatten().tolist(),
            "cD": cD7.flatten().tolist()
        }
    })
    
    return test_cases

def main():
    print("Creating DWT test cases...")
    
    test_cases = create_test_cases()
    
    # Output to JSON file
    with open('../internal/test/testcase/dwt_test_cases.json', 'w') as f:
        json.dump(test_cases, f, indent=2)
    
    print(f"Generated {len(test_cases)} test cases in dwt_test_cases.json")
    
    # Display results
    for i, case in enumerate(test_cases):
        print(f"\nTest case {i+1}: {case['name']}")
        print(f"Input shape: {case['input']['height']}x{case['input']['width']}")
        print(f"cA shape: {len(case['expected']['cA'])} elements")
        print(f"cA values: {case['expected']['cA']}")
        print(f"cH values: {case['expected']['cH']}")
        print(f"cV values: {case['expected']['cV']}")
        print(f"cD values: {case['expected']['cD']}")


if __name__ == "__main__":
    main()
