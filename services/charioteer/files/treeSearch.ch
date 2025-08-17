// treeSearch
// Find all users with role "dev"
setq(devUsers, treeSearch(data, 'role', 'dev'))

// Find all products with price > 100
setq(expensiveProducts, treeSearch(products, 'price', 100, '>'))

// Find all users whose name contains "bob"
setq(bobUsers, treeSearch(users, 'name', 'bob', 'contains'))

// Find all users whose name starts with "A"
setq(aUsers, treeSearch(users, 'name', 'A', 'startswith'))