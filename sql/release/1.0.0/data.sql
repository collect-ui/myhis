ALTER TABLE img ADD "type" varchar(255);
update img
set type= 'img'
