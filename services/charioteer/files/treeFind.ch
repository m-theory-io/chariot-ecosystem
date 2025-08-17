// treeFind
setq(products, parseJSON('[
    {"id": 1, "name": "Laptop", "price": 999},
    {"id": 2, "name": "Mouse", "price": 25},
    {"id": 3, "name": "Monitor", "price": 300}
]', 'products'))

setq(expensiveProducts, treeFind(products, 'price', 999))
