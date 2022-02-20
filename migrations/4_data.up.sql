UPDATE users SET id = '00000000-0000-0000-0000-000000000000' WHERE email='fahrulrozi1288@gmail.com';

INSERT INTO categories (id, name, description, user_id, created_at, updated_at) VALUES
('00000000-0000-0000-0000-000000000000','Housing','Housing Needs', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000001','Transportation','Transportation Needs', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000002','Medical','Medical Needs', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000003','Utilities','Utilities Needs', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000004','Personal','Personal Needs', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO products (id, category_id, name, description, user_id, created_at, updated_at) VALUES
('00000000-0000-0000-0000-000000000000','00000000-0000-0000-0000-000000000000','Mortgage/Rent','Used to buy or refinance a home', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000001','00000000-0000-0000-0000-000000000000','Maintenance','Regular check up', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000002','00000000-0000-0000-0000-000000000001','Repairs','Getting the car fixed', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000003','00000000-0000-0000-0000-000000000001','Gas','Car gas', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000004','00000000-0000-0000-0000-000000000002','Glasses/Contacts','Self explanatory', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000005','00000000-0000-0000-0000-000000000002','Dental','Dental check up', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000006','00000000-0000-0000-0000-000000000003','Electric','Self explanatory', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000007','00000000-0000-0000-0000-000000000003','Water','Self explanatory', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000008','00000000-0000-0000-0000-000000000004','Clothing','Fashion style', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP),
('00000000-0000-0000-0000-000000000009','00000000-0000-0000-0000-000000000004','Gym Membership','Experience the best gym facilities', '00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP);

INSERT INTO expenses (product_id, date, user_id, created_at, updated_at, currency, amount) VALUES
('00000000-0000-0000-0000-000000000000','2022-02-01','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'USD',100.05),
('00000000-0000-0000-0000-000000000001','2022-02-02','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'USD',88.51),
('00000000-0000-0000-0000-000000000002','2022-02-03','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'USD',44),
('00000000-0000-0000-0000-000000000003','2022-02-04','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'IDR',1000000),
('00000000-0000-0000-0000-000000000004','2022-02-05','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'IDR',2000000),
('00000000-0000-0000-0000-000000000005','2022-02-06','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'IDR',550000),
('00000000-0000-0000-0000-000000000006','2022-02-07','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'IDR',510000.00),
('00000000-0000-0000-0000-000000000007','2022-02-08','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'IDR',11225200),
('00000000-0000-0000-0000-000000000008','2022-02-09','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'USD',33.35),
('00000000-0000-0000-0000-000000000009','2022-02-10','00000000-0000-0000-0000-000000000000', CURRENT_TIMESTAMP, CURRENT_TIMESTAMP, 'USD',41.23);