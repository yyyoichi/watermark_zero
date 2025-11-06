import numpy as np
import json
from numpy.linalg import svd

def create_svd_test_cases():
    """Create SVD test cases and output as JSON"""
    test_cases = []
    
    # Test case 1: Simple 2x2 matrix
    data1 = np.array([
        [3.0, 1.0],
        [1.0, 3.0]
    ], dtype=np.float64)
    
    U1, s1, Vt1 = svd(data1)
    
    test_cases.append({
        "name": "2x2_simple",
        "input": {
            "data": data1.flatten().tolist(),
            "width": 2,
            "height": 2
        },
        "expected": {
            "singular_values": s1.tolist(),
            "u": U1.flatten().tolist(),
            "vt": Vt1.flatten().tolist()
        }
    })
    
    # Test case 2: 3x3 matrix
    data2 = np.array([
        [1.0, 2.0, 3.0],
        [4.0, 5.0, 6.0],
        [7.0, 8.0, 9.0]
    ], dtype=np.float64)
    
    U2, s2, Vt2 = svd(data2)
    
    test_cases.append({
        "name": "3x3_sequential",
        "input": {
            "data": data2.flatten().tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "singular_values": s2.tolist(),
            "u": U2.flatten().tolist(),
            "vt": Vt2.flatten().tolist()
        }
    })
    
    # Test case 3: 3x2 rectangular matrix (tall)
    data3 = np.array([
        [1.0, 2.0],
        [3.0, 4.0],
        [5.0, 6.0]
    ], dtype=np.float64)
    
    U3, s3, Vt3 = svd(data3)
    
    test_cases.append({
        "name": "3x2_tall_rectangular",
        "input": {
            "data": data3.flatten().tolist(),
            "width": 2,
            "height": 3
        },
        "expected": {
            "singular_values": s3.tolist(),
            "u": U3.flatten().tolist(),
            "vt": Vt3.flatten().tolist()
        }
    })
    
    # Test case 4: 2x3 rectangular matrix (wide)
    data4 = np.array([
        [1.0, 2.0, 3.0],
        [4.0, 5.0, 6.0]
    ], dtype=np.float64)
    
    U4, s4, Vt4 = svd(data4)
    
    test_cases.append({
        "name": "2x3_wide_rectangular",
        "input": {
            "data": data4.flatten().tolist(),
            "width": 3,
            "height": 2
        },
        "expected": {
            "singular_values": s4.tolist(),
            "u": U4.flatten().tolist(),
            "vt": Vt4.flatten().tolist()
        }
    })
    
    # Test case 5: Identity matrix
    data5 = np.eye(3, dtype=np.float64)
    
    U5, s5, Vt5 = svd(data5)
    
    test_cases.append({
        "name": "3x3_identity",
        "input": {
            "data": data5.flatten().tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "singular_values": s5.tolist(),
            "u": U5.flatten().tolist(),
            "vt": Vt5.flatten().tolist()
        }
    })
    
    # Test case 6: Diagonal matrix
    data6 = np.array([
        [5.0, 0.0, 0.0],
        [0.0, 3.0, 0.0],
        [0.0, 0.0, 1.0]
    ], dtype=np.float64)
    
    U6, s6, Vt6 = svd(data6)
    
    test_cases.append({
        "name": "3x3_diagonal",
        "input": {
            "data": data6.flatten().tolist(),
            "width": 3,
            "height": 3
        },
        "expected": {
            "singular_values": s6.tolist(),
            "u": U6.flatten().tolist(),
            "vt": Vt6.flatten().tolist()
        }
    })
    
    # Test case 7: Random matrix
    np.random.seed(42)  # for reproducible results
    data7 = np.random.rand(2, 4).astype(np.float64)
    
    U7, s7, Vt7 = svd(data7)
    
    test_cases.append({
        "name": "2x4_random",
        "input": {
            "data": data7.flatten().tolist(),
            "width": 4,
            "height": 2
        },
        "expected": {
            "singular_values": s7.tolist(),
            "u": U7.flatten().tolist(),
            "vt": Vt7.flatten().tolist()
        }
    })
    
    return test_cases

def main():
    print("Creating SVD test cases...")
    
    test_cases = create_svd_test_cases()
    
    # Output to JSON file
    with open('../internal/test/testcase/svd_test_cases.json', 'w') as f:
        json.dump(test_cases, f, indent=2)
    
    print(f"Generated {len(test_cases)} test cases in svd_test_cases.json")
    
    # Display results
    for i, case in enumerate(test_cases):
        print(f"\nTest case {i+1}: {case['name']}")
        print(f"Input shape: {case['input']['height']}x{case['input']['width']}")
        input_data = case['input']['data']
        s_values = case['expected']['singular_values']
        
        # Display first few input values and singular values
        print("Input values (first 6):", [f"{x:.6f}" for x in input_data[:6]])
        print("Singular values:", [f"{x:.6f}" for x in s_values])

if __name__ == "__main__":
    main()