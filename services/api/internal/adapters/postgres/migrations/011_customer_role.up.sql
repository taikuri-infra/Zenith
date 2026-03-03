-- Convert existing non-admin registrations to customer role
UPDATE users SET role = 'customer' WHERE role = 'developer';
