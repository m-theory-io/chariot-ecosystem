// offerVariable library
declareGlobal(offerVars, 'M', mapValue(
    'monthly', offerVar(1000, 'currency'),
    'downpayment', offerVar(1000, 'currency'),
    'discount', offerVar(0.20, 'percentage'),
    'interest', offerVar(0.12, 'percentage'),
    'term', offerVar(12, 'int'),
    'period' 'month',
    'name', 'Just a Test')
)
declare(offer, 'J')
setAttributes(offer, offerVars)
declare(profile, 'J', parseJSON('{"name": "Jane Swinney"}'))
declare(text, 'S', 'Name: {name}, Monthly: {monthly}, Downpayment: {downpayment}, Discount: {discount}, Interest: {interest}, Term: {term}, Period: {period}s')
merge(text, offerVars, [profile, offerVars]) 