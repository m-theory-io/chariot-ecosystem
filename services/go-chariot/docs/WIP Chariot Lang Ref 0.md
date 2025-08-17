Excellent! Let's create comprehensive documentation for the Chariot system. I'll start with the Chariot Language Reference, which is the foundation users need to understand.

# Chariot Language Reference

## Table of Contents

1. Introduction
2. Language Fundamentals
3. Data Types
4. Variables and Scope
5. Functions
6. Control Flow
7. Tree Structures
8. Built-in Functions
9. Agent Serialization
10. Security and Cryptography
11. Examples

---

## Introduction

Chariot is a domain-specific language designed for building intelligent decision-making agents in financial services and other enterprise applications. It combines functional programming concepts with tree-based data structures and built-in security features.

### Key Features

- **Functional programming** with immutable data structures
- **Tree-based agents** for complex decision logic
- **Built-in cryptography** and secure serialization
- **Azure Key Vault integration** for enterprise security
- **Real-time evaluation** with immediate feedback
- **Financial industry compliance** ready

### Philosophy

Chariot emphasizes:
- **Clarity** - Code should be readable and self-documenting
- **Security** - Cryptographic operations are first-class citizens
- **Composability** - Build complex agents from simple components
- **Immutability** - Data structures don't change, they evolve

---

## Language Fundamentals

### Syntax Overview

Chariot uses **prefix notation** (also called Polish notation) where operators come before their operands:

```chariot
// Arithmetic
add(5, 3)          // 8
multiply(add(2, 3), 4)  // 20

// Comparison
greater(10, 5)     // true
equal(x, y)        // true/false

// Function calls
getAttribute(profile, 'age')
```

### Comments

```chariot
// Single line comment

/* Multi-line
   comment */
```

### Case Sensitivity

Chariot is **case-sensitive**:
- `myVariable` and `MyVariable` are different
- Function names are lowercase by convention
- Constants use UPPER_CASE

---

## Data Types

### Numbers

All numbers are floating-point internally:

```chariot
42          // Integer literal
3.14159     // Decimal literal
-17         // Negative number
1.23e-4     // Scientific notation
```

### Strings

Strings are enclosed in single or double quotes:

```chariot
'Hello, World!'
"Financial Services"
'Multi-line\nstring with\tescapes'
```

**Escape Sequences:**
- `\n` - Newline
- `\t` - Tab
- `\'` - Single quote
- `\"` - Double quote
- `\\` - Backslash

### Booleans

```chariot
true
false
```

### Lists

Ordered collections of values:

```chariot
list(1, 2, 3, 4, 5)
list('apple', 'banana', 'cherry')
list()  // Empty list
```

### Trees

Hierarchical structures with attributes:

```chariot
// Create a tree node
tree('Profile', 
    attribute('name', 'John Doe'),
    attribute('age', 35),
    attribute('balance', 150000.50)
)
```

### Functions

First-class values that can be stored and passed around:

```chariot
// Define a function
func(x, y) { add(x, y) }

// Function with complex logic
func(profile) {
    if(greater(getAttribute(profile, 'age'), 18),
       getAttribute(profile, 'creditScore'),
       0)
}
```

---

## Variables and Scope

### Variable Declaration

Use `setq` to bind values to variables:

```chariot
setq(customerAge, 35)
setq(interestRate, 4.5)
setq(approvalRule, func(score) { greater(score, 700) })
```

### Variable Access

Simply use the variable name:

```chariot
setq(x, 10)
setq(y, 20)
setq(sum, add(x, y))  // sum = 30
```

### Scope Rules

Chariot uses **lexical scoping**:

```chariot
setq(globalVar, 'I am global')

setq(myFunction, func(localVar) {
    // localVar is only visible inside this function
    // globalVar is accessible here
    concatenate(globalVar, localVar)
})
```

---

## Functions

### Function Definition

```chariot
// Basic function
func(parameter1, parameter2) { 
    // function body
    add(parameter1, parameter2)
}

// Function with conditional logic
func(customer) {
    if(greater(getAttribute(customer, 'income'), 50000),
       'approved',
       'denied')
}
```

### Function Application

```chariot
setq(addFunction, func(a, b) { add(a, b) })
call(addFunction, 5, 3)  // Returns 8

// Direct application
setq(result, func(x) { multiply(x, 2) })(10)  // Returns 20
```

### Higher-Order Functions

Functions can take other functions as parameters:

```chariot
setq(applyRule, func(rule, data) {
    call(rule, data)
})

setq(ageRule, func(profile) { 
    greater(getAttribute(profile, 'age'), 21) 
})

call(applyRule, ageRule, customerProfile)
```

---

## Control Flow

### Conditional Expressions

```chariot
// Basic if-then-else
if(condition, thenValue, elseValue)

// Example
if(greater(balance, 1000),
   'sufficient funds',
   'insufficient funds')

// Nested conditionals
if(greater(age, 65),
   'senior',
   if(greater(age, 18),
      'adult',
      'minor'))
```

### Logical Operations

```chariot
// Boolean logic
and(condition1, condition2)
or(condition1, condition2)
not(condition)

// Example: Credit approval logic
and(greater(creditScore, 700),
    greater(income, 40000),
    less(debtToIncomeRatio, 0.4))
```

---

## Tree Structures

### Creating Trees

```chariot
// Customer profile tree
setq(customer, 
    tree('CustomerProfile',
        attribute('personalInfo',
            tree('PersonalInfo',
                attribute('name', 'Alice Johnson'),
                attribute('age', 32),
                attribute('ssn', '***-**-1234')
            )
        ),
        attribute('financial',
            tree('FinancialInfo',
                attribute('income', 75000),
                attribute('creditScore', 780),
                attribute('accounts', list('checking', 'savings', 'investment'))
            )
        )
    )
)
```

### Tree Navigation

```chariot
// Get attributes
getAttribute(customer, 'personalInfo')
getAttribute(getAttribute(customer, 'personalInfo'), 'name')

// Get children
getChildren(customer)
getChildAt(customer, 0)

// Tree metadata
getNodeName(customer)         // 'CustomerProfile'
getAttributeNames(customer)   // list('personalInfo', 'financial')
```

### Tree Modification

Trees are immutable, so modifications create new trees:

```chariot
// Add attribute
setq(updatedCustomer, 
    setAttribute(customer, 'lastUpdate', currentTime()))

// Add child
setq(expandedTree,
    addChild(customer, 
        tree('RiskAssessment',
            attribute('riskLevel', 'low'),
            attribute('factors', list('high_income', 'good_credit'))
        )
    )
)
```

---

## Built-in Functions

### Arithmetic

```chariot
add(a, b)           // Addition
subtract(a, b)      // Subtraction  
multiply(a, b)      // Multiplication
divide(a, b)        // Division
modulo(a, b)        // Remainder
power(base, exp)    // Exponentiation
abs(n)              // Absolute value
sqrt(n)             // Square root
```

### Comparison

```chariot
equal(a, b)         // Equality
notEqual(a, b)      // Inequality
greater(a, b)       // Greater than
greaterEqual(a, b)  // Greater than or equal
less(a, b)          // Less than
lessEqual(a, b)     // Less than or equal
```

### String Operations

```chariot
concatenate(str1, str2)     // String concatenation
substring(str, start, len)  // Extract substring
length(str)                 // String length
toUpperCase(str)           // Convert to uppercase
toLowerCase(str)           // Convert to lowercase
contains(str, substr)      // Check if contains substring
```

### List Operations

```chariot
list(item1, item2, ...)    // Create list
append(list, item)         // Add item to end
prepend(list, item)        // Add item to beginning
first(list)                // Get first element
rest(list)                 // Get all but first
length(list)               // List length
contains(list, item)       // Check membership
map(function, list)        // Apply function to each element
filter(predicate, list)    // Filter elements
```

### Type Checking

```chariot
isNumber(value)
isString(value)
isBoolean(value)
isList(value)
isTree(value)
isFunction(value)
```

### Utility Functions

```chariot
print(value)               // Print to console
toConsole(value)          // Debug output
currentTime()             // Current timestamp
randomNumber(min, max)    // Random number generation
```

---

## Agent Serialization

### JSON Serialization

```chariot
// Save agent to JSON file
treeSave(agent, 'trading_agent.json')

// Load agent from JSON file
setq(loadedAgent, treeLoad('trading_agent.json'))

// Convert to JSON string
setq(jsonString, treeToJson(agent))

// Parse from JSON string
setq(parsedAgent, treeFromJson(jsonString))
```

### Secure Binary Serialization

For production financial applications:

```chariot
// Save with enterprise security
treeSaveSecure(agent, 'secure_agent.dat', 
               'encryption-key-id', 
               'signing-key-id',
               'Goldman Sachs Trading Model v2.1')

// Load with verification
setq(secureAgent, treeLoadSecure('secure_agent.dat',
                                'decryption-key-id',
                                'verification-key-id'))

// Validate integrity without loading
setq(isValid, treeValidateSecure('secure_agent.dat'))
```

---

## Security and Cryptography

### Encryption

```chariot
// Encrypt sensitive data
setq(encrypted, encrypt('aes-key-id', 'sensitive customer data'))

// Decrypt data
setq(decrypted, decrypt('aes-key-id', encrypted))
```

### Digital Signatures

```chariot
// Sign document
setq(signature, sign('rsa-signing-key', documentText))

// Verify signature
setq(isValid, verify('rsa-verification-key', documentText, signature))
```

### Hashing

```chariot
// Calculate SHA-256 hash
setq(hash, hash256('data to hash'))

// Generate random data
setq(randomBytes, randomBytes(32))  // 32 random bytes
setq(aesKey, generateKey(32))       // 256-bit AES key
```

**Security Note:** All cryptographic operations use Azure Key Vault for key management. Keys are referenced by ID, never stored in code.

---

## Examples

### Example 1: Customer Credit Approval

```chariot
// Define credit approval function
setq(creditApproval, func(customer) {
    setq(score, getAttribute(customer, 'creditScore'))
    setq(income, getAttribute(customer, 'income'))
    setq(debtRatio, getAttribute(customer, 'debtToIncomeRatio'))
    
    if(and(greater(score, 700),
           greater(income, 50000),
           less(debtRatio, 0.4)),
       'APPROVED',
       'DENIED')
})

// Create customer profile
setq(customer1, 
    tree('Customer',
        attribute('name', 'John Smith'),
        attribute('creditScore', 750),
        attribute('income', 65000),
        attribute('debtToIncomeRatio', 0.25)
    )
)

// Apply approval logic
setq(decision, call(creditApproval, customer1))
toConsole(decision)  // Outputs: APPROVED
```

### Example 2: Investment Portfolio Analysis

```chariot
// Define risk assessment function
setq(assessRisk, func(portfolio) {
    setq(stocks, getAttribute(portfolio, 'stockPercentage'))
    setq(bonds, getAttribute(portfolio, 'bondPercentage'))
    setq(cash, getAttribute(portfolio, 'cashPercentage'))
    
    if(greater(stocks, 80),
       'HIGH_RISK',
       if(greater(stocks, 50),
          'MEDIUM_RISK',
          'LOW_RISK'))
})

// Create portfolio
setq(portfolio, 
    tree('Portfolio',
        attribute('stockPercentage', 65),
        attribute('bondPercentage', 25),
        attribute('cashPercentage', 10),
        attribute('totalValue', 1500000)
    )
)

// Assess risk
setq(riskLevel, call(assessRisk, portfolio))
```

### Example 3: Complex Decision Tree

```chariot
// Loan underwriting decision tree
setq(loanAgent,
    tree('LoanUnderwriting',
        // Primary income check
        tree('IncomeVerification',
            attribute('minimumIncome', 40000),
            attribute('rule', func(applicant) {
                greater(getAttribute(applicant, 'income'), 40000)
            })
        ),
        
        // Credit history analysis
        tree('CreditAnalysis',
            attribute('minimumScore', 650),
            attribute('rule', func(applicant) {
                setq(score, getAttribute(applicant, 'creditScore'))
                setq(history, getAttribute(applicant, 'creditHistory'))
                
                and(greater(score, 650),
                    greater(length(history), 2))
            })
        ),
        
        // Final decision logic
        attribute('finalDecision', func(applicant) {
            setq(incomeCheck, call(
                getAttribute(getChildAt(loanAgent, 0), 'rule'),
                applicant))
            
            setq(creditCheck, call(
                getAttribute(getChildAt(loanAgent, 1), 'rule'),
                applicant))
                
            if(and(incomeCheck, creditCheck),
               concatenate('APPROVED - Rate: ', 
                          calculateRate(applicant)),
               'DENIED')
        })
    )
)

// Save the agent securely
treeSaveSecure(loanAgent, 'loan_agent.secure',
               'loan-encryption-key',
               'loan-signing-key',
               'Bank Loan Underwriting Model v3.2')
```

### Example 4: Real-time Fraud Detection

```chariot
// Fraud detection agent
setq(fraudDetector, func(transaction) {
    // Multiple risk factors
    setq(amount, getAttribute(transaction, 'amount'))
    setq(location, getAttribute(transaction, 'location'))
    setq(timeOfDay, getAttribute(transaction, 'hour'))
    setq(merchantType, getAttribute(transaction, 'merchantType'))
    
    // Risk scoring
    setq(amountRisk, if(greater(amount, 5000), 30, 0))
    setq(locationRisk, if(equal(location, 'foreign'), 25, 0))
    setq(timeRisk, if(or(less(timeOfDay, 6), greater(timeOfDay, 23)), 15, 0))
    setq(merchantRisk, if(contains(list('gambling', 'crypto'), merchantType), 20, 0))
    
    setq(totalRisk, add(add(amountRisk, locationRisk), 
                        add(timeRisk, merchantRisk)))
    
    // Decision based on risk score
    if(greater(totalRisk, 50),
       'BLOCK_TRANSACTION',
       if(greater(totalRisk, 25),
          'REQUIRE_ADDITIONAL_AUTH',
          'APPROVE'))
})
```

---

## Best Practices

### 1. Naming Conventions

```chariot
// Variables: camelCase
setq(customerProfile, ...)
setq(interestRate, ...)

// Functions: descriptive verbs
setq(calculateInterest, func(...) { ... })
setq(validateCreditScore, func(...) { ... })

// Constants: UPPER_CASE
setq(MAX_LOAN_AMOUNT, 1000000)
setq(MIN_CREDIT_SCORE, 600)
```

### 2. Code Organization

```chariot
// Group related functions
setq(creditUtils, 
    tree('CreditUtilities',
        attribute('scoreToRating', func(score) { ... }),
        attribute('calculateRate', func(score, term) { ... }),
        attribute('approvalLogic', func(applicant) { ... })
    )
)
```

### 3. Error Handling

```chariot
// Defensive programming
setq(safeGetAttribute, func(tree, name, defaultValue) {
    if(hasAttribute(tree, name),
       getAttribute(tree, name),
       defaultValue)
})
```

### 4. Security

```chariot
// Never hardcode sensitive data
// BAD: setq(apiKey, 'sk-1234567890abcdef')
// GOOD: Use Key Vault references
setq(encryptedData, encrypt('api-key-vault-id', sensitiveData))

// Always validate inputs
setq(validateInput, func(data) {
    and(isTree(data),
        hasAttribute(data, 'requiredField'),
        greater(length(getAttribute(data, 'requiredField')), 0))
})
```

---

This completes the Chariot Language Reference. Would you like me to proceed with the Chariot Server documentation next?