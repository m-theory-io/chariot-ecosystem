setq(myTree, parseJSON('[10,34,67]', 'numbers'))
treeWalk(myTree, func(myTree) Value {
    setq(tname, concat('_', getName(myTree)))
    getAttribute(myTree, tname)
    log('visiting', getName(myTree))
})

