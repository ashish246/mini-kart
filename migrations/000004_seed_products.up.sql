-- Seed sample products (idempotent - uses ON CONFLICT DO NOTHING)

-- Waffles
INSERT INTO products (id, name, price, category) VALUES
    ('1', 'Classic Belgian Waffle', 8.95, 'Waffle'),
    ('2', 'Chocolate Chip Waffle', 9.95, 'Waffle'),
    ('3', 'Strawberry Waffle', 10.95, 'Waffle')
ON CONFLICT (id) DO NOTHING;

-- Sandwiches
INSERT INTO products (id, name, price, category) VALUES
    ('4', 'Chicken Caesar Sandwich', 12.50, 'Sandwich'),
    ('5', 'Turkey Club Sandwich', 11.95, 'Sandwich'),
    ('6', 'Veggie Delight Sandwich', 9.95, 'Sandwich')
ON CONFLICT (id) DO NOTHING;

-- Beverages
INSERT INTO products (id, name, price, category) VALUES
    ('7', 'Fresh Orange Juice', 4.50, 'Beverage'),
    ('8', 'Cappuccino', 5.25, 'Beverage'),
    ('9', 'Iced Latte', 5.75, 'Beverage'),
    ('10', 'Sparkling Water', 3.00, 'Beverage')
ON CONFLICT (id) DO NOTHING;

-- Desserts
INSERT INTO products (id, name, price, category) VALUES
    ('11', 'Chocolate Brownie', 6.95, 'Dessert'),
    ('12', 'Tiramisu', 7.95, 'Dessert'),
    ('13', 'Cheesecake', 7.50, 'Dessert')
ON CONFLICT (id) DO NOTHING;
