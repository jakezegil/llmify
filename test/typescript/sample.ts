// A simple TypeScript file for testing refactoring

function calculateSum(a: number, b: number): number {
  let result = a + b;
  return result;
}

function calculateDifference(a: number, b: number): number {
  let result = a - b;
  return result;
}

function calculateProduct(a: number, b: number): number {
  let result = a * b;
  return result;
}

// Example usage
let num1 = 10;
let num2 = 5;

console.log(`Sum: ${calculateSum(num1, num2)}`);
console.log(`Difference: ${calculateDifference(num1, num2)}`);
console.log(`Product: ${calculateProduct(num1, num2)}`); 